package engine

import (
	"fmt"
	"os"
)

// More script commands (O03C), and the ones still to port: those step over
// their operands (parse_command) and say so on stderr.

func (e *Engine) unported(name string) {
	fmt.Fprintf(os.Stderr, "cotab: script command %d (%s) at %d not ported yet\n", e.sc.command, name, e.sc.pc)
	e.skipCommand()
}

func (e *Engine) cmdCombat()       { e.unported("COMBAT") }
func (e *Engine) cmdRob()          { e.unported("ROB") }
func (e *Engine) cmdParlay()       { e.unported("PARLAY") }
func (e *Engine) cmdDamage()       { e.unported("DAMAGE") }
func (e *Engine) cmdClock()        { e.unported("CLOCK") }
func (e *Engine) cmdAddNPC()       { e.unported("ADD NPC") }
func (e *Engine) cmdProgram()      { e.unported("PROGRAM") }
func (e *Engine) cmdWho()          { e.unported("WHO") }
func (e *Engine) cmdProtection()   { e.unported("PROTECTION") }
func (e *Engine) cmdClearBox()     { e.unported("CLEAR BOX") }

// cmdOnGotoGosub = O03C:1AE5 (ON GOTO, ON GOSUB): jump to the n-th of the
// addresses that follow.
func (e *Engine) cmdOnGotoGosub() {
	s := &e.sc
	e.readArgs(2)
	n := byte(e.arg(1))
	count := byte(e.arg(2))
	s.pc--
	e.readArgs(int(count))
	for k := byte(0); k < count; k++ {
		if k == n {
			if s.command == cmdOnGoto {
				s.pc = varAddr(s.argLo[k+1], s.argHi[k+1])
			} else {
				e.scriptCall(int(k + 1))
			}
			return
		}
	}
}

// cmdCheckParty = O03C:1454: the party's affect, thief skill or movement
// census into four variables (found / lowest, highest, average, count).
func (e *Engine) cmdCheckParty() {
	s := &e.sc
	e.readArgs(6)
	var what uint16
	if s.argType[1] == argVarByte {
		what = varAddr(s.argLo[1], s.argHi[1])
	} else {
		what = e.arg(1)
	}
	affect := byte(e.arg(2))
	dst := [4]uint16{
		varAddr(s.argLo[3], s.argHi[3]), varAddr(s.argLo[4], s.argHi[4]),
		varAddr(s.argLo[5], s.argHi[5]), varAddr(s.argLo[6], s.argHi[6]),
	}
	var total uint16
	var count byte
	lowest, highest, average := byte(255), byte(0), byte(0)
	store := func() { // O03C:13F6
		e.setVar(dst[0], uint16(lowest))
		e.setVar(dst[1], uint16(highest))
		e.setVar(dst[2], uint16(average))
		e.setVar(dst[3], signed(count))
	}
	census := func(field func(*Character) byte) {
		for _, c := range e.party {
			count++
			v := field(c)
			if v < lowest {
				lowest = v
			}
			if v > highest {
				highest = v
			}
			total += uint16(v)
		}
		average = byte(int16(total) / int16(count))
		store()
	}
	switch v := int16(what - 32767); {
	case v == -32767:
		found := false
		for _, c := range e.party {
			if found = e.FindAffect(c, affect) != nil; found {
				break
			}
		}
		lowest, highest, average = 0, 0, 0
		if found {
			lowest = 1
		}
		store()
	case v >= cvThiefFirst && v <= cvThiefLast:
		k := v - cvThiefFirst
		census(func(c *Character) byte { return c.Thief[k] })
	case v == cvMovement:
		census(func(c *Character) byte { return c.Move })
	}
}

// cvMovement: CHECK PARTY's movement census (address 32767 + 159).
const cvMovement = 159

// cmdPartySurprise = O03C:167E: whether a ranger is in the party (no one
// is surprised then); the second result is always 0.
func (e *Engine) cmdPartySurprise() {
	e.readArgs(2)
	ranger := byte(0)
	for _, c := range e.party {
		if c.Class == classRanger || c.Class == classRangerMage {
			ranger = 1
		}
	}
	e.setVar(varAddr(e.sc.argLo[1], e.sc.argHi[1]), uint16(ranger))
	e.setVar(varAddr(e.sc.argLo[2], e.sc.argHi[2]), 0)
}

// Class numbers PARTY SURPRISE looks for.
const (
	classRanger     = 4
	classRangerMage = 10
)

// surpriseResult: the global SURPRISE stores 0 none, 1 the party, 2 the
// monsters, 3 both.
const surpriseResult = 715

// cmdSurprise = O03C:171E: d6 rolls for both sides against 2 + the
// modifiers.
func (e *Engine) cmdSurprise() {
	e.readArgs(4)
	var a [5]byte
	for k := 1; k <= 4; k++ {
		a[k] = byte(e.arg(k))
	}
	party := int8(a[2] + 2 - a[1])
	monsters := int8(a[4] + 2 - a[3])
	r1 := int8(e.Dice(1, 6))
	r2 := int8(e.Dice(1, 6))
	var r uint16
	if r1 <= party {
		if r2 <= monsters {
			r = 3
		} else {
			r = 1
		}
	}
	if r2 <= monsters {
		r = 2
	}
	e.setVar(surpriseResult, r)
}

// cmdSpriteOff = O03C:2CB6.
func (e *Engine) cmdSpriteOff() {
	s := &e.sc
	s.pc++
	if s.spriteOn {
		e.viewReset = true
		e.drawView()
		s.spriteOn = false
		s.pictureOn = false
	}
}

// cmdFindItem = O03C:2856: equal when someone in the party carries an
// item of the type.
func (e *Engine) cmdFindItem() {
	e.readArgs(1)
	typ := byte(e.arg(1))
	e.sc.flags = [6]bool{}
	e.sc.flags[1] = true
	for _, c := range e.party {
		for _, it := range c.Items {
			if it.Type == typ {
				e.sc.flags[0], e.sc.flags[1] = true, false
				return
			}
		}
	}
}

// cmdDelay = O03C:290B.
func (e *Engine) cmdDelay() {
	e.sc.pc++
	e.gameDelay()
}

// cmdPrintReturn = O03C:2D0F: the text cursor to the next line.
func (e *Engine) cmdPrintReturn() {
	s := &e.sc
	s.pc++
	e.msgCol = textLeft
	e.msgRow++
	e.textNewLine = false
}

// cmdSpell = O03C:2E33: who in the party has memorized the spell: its
// slot (255 none) and the member's place.
func (e *Engine) cmdSpell() {
	s := &e.sc
	e.readArgs(3)
	spell := byte(e.arg(1))
	slotDst := varAddr(s.argLo[2], s.argHi[2])
	whoDst := varAddr(s.argLo[3], s.argHi[3])
	slot, who := 1, byte(0)
	found := false
	for i, c := range e.party {
		slot = 1
		for !found {
			if c.Memorized[slot-1] == spell { // record offset 30 + slot
				found = true
			} else if slot <= maxSpellSlot {
				slot++
			} else {
				break
			}
		}
		if found {
			break
		}
		if i+1 < len(e.party) {
			who++
		}
	}
	if slot > maxSpellSlot {
		slot = 255
	}
	e.setVar(slotDst, uint16(slot))
	e.setVar(whoDst, uint16(who))
}

// maxSpellSlot: SPELL looks through the 100 bytes after the record's
// offset 30 (the memorized spells and what follows them).
const maxSpellSlot = 100

// cmdDump = O03C:3271: the selected member leaves the party.
func (e *Engine) cmdDump() {
	e.sc.pc++
	e.freeSelected()
	e.sc.savedSelected = e.selected()
	e.playerSummary(e.selected())
}

// cmdFindSpecial = O03C:32A5: equal when the selected member has the affect.
func (e *Engine) cmdFindSpecial() {
	e.sc.flags = [6]bool{}
	e.readArgs(1)
	if e.FindAffect(e.selected(), byte(e.arg(1))) != nil {
		e.sc.flags[0] = true
		return
	}
	e.sc.flags[1] = true
}

// cmdDestroyItems = O03C:32F9: every item of the type leaves the party.
func (e *Engine) cmdDestroyItems() {
	e.readArgs(1)
	typ := byte(e.arg(1))
	for _, c := range e.party {
		kept := c.Items[:0]
		for _, it := range c.Items {
			if it.Type != typ {
				kept = append(kept, it)
			}
		}
		c.Items = kept
		e.ApplyItems(c)
	}
}

// scriptDuel = O04C:1B9A duel (win): Tyranthraxus' duel (M7).
func (e *Engine) scriptDuel(win bool) {}

// cmdVerticalMenu = O03C:0EE9 (dst, text, count, items...): the text in the
// text window and the items as a list under it; the chosen item's index
// goes to dst.
func (e *Engine) cmdVerticalMenu() {
	s := &e.sc
	e.waitedKeyReset()
	redraw := true
	e.readArgs(3)
	dst := varAddr(s.argLo[1], s.argHi[1])
	text := s.text[1]
	n := int(byte(e.arg(3)))
	s.pc--
	e.readArgs(n)
	list := newList(n)
	e.msgCol, e.msgRow = textLeft, textTop
	e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourGreen, true, text)
	for i, node := range list {
		node.Text = s.text[i+1]
		node.Flag = 0
	}
	index := 0
	var sel *ListNode
	e.ListMenu("", "", colourLightMagenta, colourGreen, colourWhite, textLeft, e.msgRow+1, textRight, textBottom,
		list, false, &redraw, &index, &sel)
	e.setVar(dst, uint16(byte(index)))
	e.ClearCells(textLeft, textTop, textRight, textBottom)
}

// waitedKeyReset: the next command input clears the text window again (b8B80).
func (e *Engine) waitedKeyReset() { e.sc.waitedKey = false }

// maxCombatants: LOAD MONSTER stops adding at this many monsters.
const maxCombatants = 63

// cmdLoadMonster = O03C:0465 (monster, count, icon): count copies of the
// monster join the end of the party list for the coming fight, with the
// combat icon in the next icon slot. Copies after the first get their item
// and affect lists in reverse order (each node is pushed on the head).
func (e *Engine) cmdLoadMonster() {
	s := &e.sc
	saved := e.partySel
	e.readArgs(3)
	if s.monsterCount >= maxCombatants {
		return
	}
	first := e.loadMonsterRecord(byte(e.arg(1)))
	count := byte(e.arg(2))
	if count == 0 {
		count = 1
	}
	icon := byte(e.arg(3))
	e.LoadIconPic("CPIC", int(icon), int(s.iconSlot))
	first.Order = s.iconSlot
	e.party = append(e.party, first)
	s.monsterCount++
	for n := byte(2); n <= count && s.monsterCount < maxCombatants; n++ {
		c := &Character{charRecord: first.charRecord}
		c.Order = s.iconSlot
		for k := len(first.Items) - 1; k >= 0; k-- {
			it := *first.Items[k]
			c.Items = append(c.Items, &it)
		}
		for k := len(first.Affects) - 1; k >= 0; k-- {
			a := *first.Affects[k]
			c.Affects = append(c.Affects, &a)
		}
		e.party = append(e.party, c)
		s.monsterCount++
	}
	s.iconSlot++
	s.monstersLoaded = true
	e.partySel = saved
}

// loadMonsterRecord = O0A1:329C (block): MON<area>CHA, SPC (affects) and
// ITM (items) blocks.
func (e *Engine) loadMonsterRecord(block byte) *Character {
	c := &Character{}
	name := "MON" + e.areaDigit()
	b := e.L.Block(name+"CHA", int(block))
	if b == nil {
		e.flashMessage("Unable to load monster", colourWhite, colourBlack)
		e.Exit()
		return c
	}
	c.setRecord(b)
	e.loadingMonster = true
	if b := e.L.Block(name+"SPC", int(block)); b != nil {
		c.Affects = readAffects(b)
	}
	if b := e.L.Block(name+"ITM", int(block)); b != nil {
		c.Items = readRecords[Item](b, itemRecSize)
	}
	e.loadingMonster = false
	e.ClearKeyboard()
	return c
}

// partySpeeds = O04C:20E5: the party's fastest and slowest movement
// (doubled by haste, halved by slow). Fixed: the original doubles for slow
// (39) and halves for haste (42), the other way round from the character
// sheet (O0AF).
func (e *Engine) partySpeeds() (fastest, slowest byte) {
	if len(e.party) > 0 {
		fastest, slowest = e.party[0].Move, e.party[0].Move
	}
	for _, c := range e.party {
		v := c.Move
		if e.FindAffect(c, affectHaste) != nil {
			v *= 2
		} else if e.FindAffect(c, affectSlow) != nil {
			v >>= 1
		}
		if v > fastest {
			fastest = v
		}
		if v < slowest {
			slowest = v
		}
	}
	return
}

// What an encounter ends in (stored to its result variable).
const (
	encounterGone   = 0 // the monsters flee
	encounterFight  = 1
	encounterFlee   = 2 // the party got away
	encounterParlay = 3
)

// Monster reactions to the party's choice (the operands 5..9).
const (
	reactFight   = 0
	reactWait    = 1
	reactFlee    = 2
	reactAdvance = 3
	reactParlay  = 4
)

// cmdEncounter = O03C:20BD (sprite, distance, picture, result, the
// reactions to COMBAT WAIT FLEE PARLAY ADVANCE, three texts, the monsters'
// and the party's run speeds): the party meets monsters and chooses.
func (e *Engine) cmdEncounter() {
	s := &e.sc
	s.inEncounter = true
	s.waitedKey = false
	e.slowText = true
	fastest, slowest := e.partySpeeds()
	_ = fastest
	e.readArgs(14)
	s.sprite = byte(e.arg(1))
	putWord(e.game, 2*gameWordEncDist, e.arg(2))
	s.closeup = byte(e.arg(3))
	result := varAddr(s.argLo[4], s.argHi[4])
	var reaction [5]byte
	for k := 0; k <= 4; k++ {
		reaction[k] = byte(e.arg(k + 5))
	}
	var texts [4]string
	for k := 1; k <= 3; k++ {
		texts[k] = s.text[k]
	}
	monsterSpeed := byte(e.arg(13))
	runSpeed := byte(e.arg(14))
	e.setEncounterDistance()
	e.showEncounter(s.sprite, s.closeup, byte(wordAt(e.game, 2*gameWordDistance)))
	for {
		animate := s.pictureOn && s.viewDrawn && wordAt(e.area, 2*areaWordFreeSteps) != 0 && e.mapMode != mapPicture
		clear := wordAt(e.area, 2*areaWordFreeSteps) != 0
		again := false
		e.msgCol, e.msgRow = textLeft, textTop
		dist := wordAt(e.game, 2*gameWordDistance)
		var t string
		switch dist {
		case 0:
			for k := 1; k <= 3; k++ {
				if t = texts[k]; t != "" {
					break
				}
			}
		case 1, 2:
			k := int(dist) + 1
			for {
				t = texts[k]
				k++
				if k > 3 {
					k = 1
				}
				if t != "" || k == int(dist)+1 {
					break
				}
			}
		}
		if t == "" {
			clear = false
		}
		e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourGreen, clear, t)
		menu := "~COMBAT ~WAIT ~FLEE ~ADVANCE"
		if dist == 0 || wordAt(e.area, 2*areaWordFreeSteps) == 0 {
			menu = "~COMBAT ~WAIT ~FLEE ~PARLAY"
		}
		choice := e.scriptMenu("", menu, colourLightMagenta, colourGreen, colourWhite, false, animate)
		if (dist == 0 || wordAt(e.area, 2*areaWordFreeSteps) == 0) && choice == 3 {
			choice = 4
		}
		if choice > 4 { // a digit that is no item: the original read past its table
			continue
		}
		set := func(v uint16) { e.setVar(result, v) }
		wait := func() {
			e.msgCol, e.msgRow = textLeft, textTop
			e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourGreen, true, "Both sides wait.")
		}
		closer := func() bool { // one step closer; false when already there
			d := wordAt(e.game, 2*gameWordDistance)
			if d == 0 {
				return false
			}
			putWord(e.game, 2*gameWordDistance, d-1)
			e.showEncounter(s.sprite, s.closeup, byte(d-1))
			return true
		}
		monstersFlee := func() {
			set(encounterGone)
			e.msgCol, e.msgRow = textLeft, textTop
			e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourGreen, true, "The monsters flee.")
		}
		switch reaction[choice] {
		case reactFight:
			if choice == 2 && slowest >= monsterSpeed {
				set(encounterFlee)
			} else {
				set(encounterFight)
			}
		case reactWait:
			switch choice {
			case 0:
				set(encounterFight)
			case 1:
				again = true
				wait()
			case 2:
				set(encounterFlee)
			case 3:
				if !closer() {
					wait()
				}
				again = true
			case 4:
				if dist := wordAt(e.game, 2*gameWordDistance); dist == 0 {
					set(encounterParlay)
				} else {
					closer()
					again = true
				}
			}
		case reactFlee:
			if choice == 0 {
				if runSpeed > fastest {
					monstersFlee()
				} else {
					set(encounterFight)
				}
			} else {
				monstersFlee()
			}
		case reactAdvance:
			switch choice {
			case 0:
				set(encounterFight)
			case 1, 3:
				if !closer() {
					wait()
				}
				again = true
			case 2:
				set(encounterFlee)
			case 4:
				if wordAt(e.game, 2*gameWordDistance) == 0 {
					set(encounterParlay)
				} else {
					closer()
					again = true
				}
			}
		case reactParlay:
			switch choice {
			case 0:
				set(encounterFight)
			case 1, 3, 4:
				if wordAt(e.game, 2*gameWordDistance) == 0 {
					set(encounterParlay)
				} else {
					closer()
					again = true
				}
			case 2:
				set(encounterFlee)
			}
		}
		if !again {
			break
		}
	}
	e.ClearRow24()
	e.slowText = false
	s.inEncounter = false
}
