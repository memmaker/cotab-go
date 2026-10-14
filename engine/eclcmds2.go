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

func (e *Engine) cmdLoadMonster()  { e.unported("LOAD MONSTER") }
func (e *Engine) cmdVerticalMenu() { e.unported("VERTICAL MENU") }
func (e *Engine) cmdCombat()       { e.unported("COMBAT") }
func (e *Engine) cmdTreasure()     { e.unported("TREASURE") }
func (e *Engine) cmdRob()          { e.unported("ROB") }
func (e *Engine) cmdEncounter()    { e.unported("ENCOUNTER") }
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
