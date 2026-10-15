package engine

// ovr23 (spells), first part (0x0000..0x100F) and the tail (0x2540..0x37B5).
// The spell effects between are in spells2.go. The spell record is
// DS:31FA + 16*spell (see spellRec): [0] class (casterLevel), [1] list
// group, [2]/[3] duration base/per level, [4]/[5] amount base/per level,
// [6] minimum 1 flag, [7] targeting kind, [8] save mode, [9] save type,
// [0xA] affect type.

import ()

// tgt / setTgt = DS:6C91 + 4*i: the spell target list (1-based). Slot 1 is
// DS:6C95, kept in e.spellTarget.
func (e *Engine) tgt(i int) *Character {
	if i == 1 {
		return e.spellTarget
	}
	return e.spellTargets[i]
}

func (e *Engine) setTgt(i int, c *Character) {
	if i == 1 {
		e.spellTarget = c
	} else {
		e.spellTargets[i] = c
	}
}

// o23_0729 = ovr23:0729 (spell): the duration of the spell in rounds (or
// turns); never 0 when the record says so.
func (e *Engine) o23_0729(spell int) int {
	s := byte(spell)
	var v byte
	if !e.usingItem {
		v = e.spellRec(s, 2) + byte(int(e.spellRec(s, 3))*e.casterLevel(spell))
	} else {
		v = e.spellRec(s, 2) + byte(int(e.spellRec(s, 3))*6)
	}
	if v == 0 && e.spellRec(s, 6) != 0 {
		v = 1
	}
	if v == 0xFF {
		v = 1
	}
	return int(v)
}

// o23_07D6 = ovr23:07D6 (spell): the amount (damage or rounds) of the
// spell; a near procedure of the overlay.
func (e *Engine) o23_07D6(spell int) int {
	var v int
	switch spell {
	case 0x28:
		v = e.Dice(1, 6) * 10
	case 0x39, 0x3D:
		v = e.Dice(5, 4)
	case 0x3B:
		v = e.Dice(1, 4)*10 + 0x28
	case 0x3F:
		if e.gameState == 5 {
			v = e.Dice(2, 10) * 10
		} else {
			v = (e.Dice(1, 10) + 10) * 10
		}
	case 0x43:
		v = 0x5A0
	default:
		v = int(e.spellRec(byte(spell), 4)) + int(e.spellRec(byte(spell), 5))*e.casterLevel(spell)
	}
	return int(int16(v))
}

// o23_08C3 = ovr23:08C3 (spell, level, call, dmg, dmgType, msg): applies
// the spell record to every target: saving throw, damage and affect.
// level 0 means the caster level; dmg is also the flag that sets the damage
// type (DS:6819); dmg is cleared for the rest when an attack misses.
func (e *Engine) o23_08C3(spell, level, call, dmg, dmgType int, msg string) {
	msg = cut(msg, 0x28)
	if dmg == 0 {
		e.rv.dmgType = 0
	} else {
		e.rv.dmgType = byte(dmgType)
	}
	if e.nSpellTargets == 0 {
		return
	}
	lvl := byte(level)
	if level == 0 {
		lvl = byte(e.casterLevel(spell))
	}
	s := byte(spell)
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		if t == nil {
			continue
		}
		saved := false
		if e.spellRec(s, 8) != 0 {
			saved = e.SavingThrow(t, int(e.spellRec(s, 9)), 0)
		}
		if e.spellRec(e.curSpell, 2) == 0xFF {
			e.ApplyItems(t)
			e.affectGroup(t, 0xB)
			if !e.AttackHits(e.selected(), t, int(t.AC[0])) {
				dmg = 0
				saved = true
			}
		}
		if dmg > 0 {
			e.damagePerson(t, dmg, e.spellRec(s, 8), saved)
		}
		if e.spellRec(s, 0xA) > 0 {
			e.isUnaffected(t, e.spellRec(s, 0xA), e.o23_07D6(spell), lvl, byte(call), e.spellRec(s, 8), saved, msg)
		}
	}
	e.rv.dmgType = 0
}

// spellTargetHook = lcall [DS:6B84].
func (e *Engine) spellTargetHook(spell, mode int) bool {
	if e.combatTargets {
		return e.chooseSpellTargets(spell, mode)
	}
	return e.spellTargetSetup(spell, mode)
}

// spellTargetSetup = ovr23:0A91 (spell, mode, var ok): fills the target
// list by the targeting kind of the spell (1 the caster, 2 a chosen member,
// 4 the whole party); ok is false for any other kind or when the choice is
// cancelled. This is slot 0 of the DS:6B84 table; mode is not used.
func (e *Engine) spellTargetSetup(spell, mode int) (ok bool) {
	if e.spellCaster == nil {
		e.spellCaster = e.selected()
	}
	e.spellTarget = e.selected()
	e.nSpellTargets = 1
	ok = true
	switch e.spellRec(byte(spell), 7) {
	case 1:
	case 2:
		e.DrawMainScreen()
		e.spellCaster = e.SelectCharacter("Cast Spell on whom", true, e.spellCaster)
		if e.spellCaster == nil {
			e.spellTarget = nil
			e.nSpellTargets = 0
			ok = false
		} else {
			e.spellTarget = e.spellCaster
		}
	case 4:
		for i, c := range e.party {
			e.setTgt(i+1, c)
		}
		if len(e.party) > 0 {
			e.nSpellTargets = len(e.party)
		}
	default:
		ok = false
	}
	return
}

// o23_0F4F = ovr23:0F4F (side, msg): drops the targets of the other side
// (and, for spell 1 in combat, those with an enemy next to them), then
// applies the spell record to the rest.
func (e *Engine) o23_0F4F(side int, msg string) {
	msg = cut(msg, 0x28)
	e.rv.areaSpell = 1
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		if t == nil || int(t.Side) != side ||
			e.curSpell == 1 && e.gameState == 5 && e.nearEnemies(t, 1) > 0 {
			e.setTgt(i, nil)
		}
	}
	e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, msg)
}

// CastSpell = O0C2:12E1 (spell, mode, announce, var quit): casts the spell
// of the selected character. announce prints "casts"; quit is set when a
// combat-only item was used up. A combat-only spell or item outside a fight
// asks to lose it / use it.
func (e *Engine) CastSpell(spell, mode, b byte, quit *bool) {
	caster := e.selected()
	ok := true
	if e.gameState != stateCombat && e.spellRec(spell, 7) == 0 {
		if !e.usingItem {
			e.DisplayString(1, 0x13, colourGreen, spellNames[int(spell)])
			e.DisplayString(1, 0x14, colourGreen, "can't be cast here...")
			if e.YesNo(0xD, 0xA, 0xF, "Lose it? ") == 'Y' {
				e.forgetSpell(spell)
			}
		} else {
			e.DisplayString(1, 0x13, colourGreen, "That Item")
			e.DisplayString(1, 0x14, colourGreen, "is a combat-only item...")
			if e.YesNo(0xD, 0xA, 0xF, "Use it? ") == 'Y' {
				*quit = true
			}
		}
		b = 0
		ok = false
	}
	if e.FindAffect(caster, affectMiscasting) != nil && e.Dice(1, 2) == 1 {
		e.castMessage(caster, "miscasts", int(spell))
		b = 0
		ok = false
	}
	if b != 0 && !e.usingItem {
		e.castMessage(caster, "casts", int(spell))
	}
	for ok {
		*quit = e.spellTargetHook(int(spell), int(mode)) // the asm passes the caller's var
		if *quit {
			ok = false
			if e.gameState == 5 {
				e.hitColours(0x12)
				x, y := e.combatX(caster), e.combatY(caster)
				dir := 0
				for !e.inArc(x, y, e.tgtX, e.tgtY, dir) {
					dir++
				}
				e.combatTurn(caster, byte(dir), 1, false)
				e.soundWait(0)
				switch spell {
				case 0x2F:
					e.PlaySound(sndSpell47)
				case 0x33:
					e.PlaySound(sndSpell51)
				default:
					e.PlaySound(sndSpell)
				}
				e.flyMissile(x, y, e.tgtX, e.tgtY, 4, 0x1E)
				e.soundWait(0xA)
				if e.combatOnScreen(caster, false) {
					e.combatTurn(caster, caster.Combat.Facing, 1, true)
					e.combatTurn(caster, caster.Combat.Facing, 0, false)
				}
			}
			e.removeInvisibility(caster)
			if !e.usingItem {
				e.forgetSpell(spell)
			}
			e.curSpell = spell
			e.spellFx[spell]()
			e.curSpell = 0
			e.rv.areaSpell = 0
		} else if e.gameState != 5 {
			ok = false
		} else {
			if mode == 0 && e.YesNo(0xE, 0xA, 0xF, "Abort Spell? ") != 'Y' {
				continue
			}
			e.BottomMessage("Spell Aborted")
			if !e.usingItem {
				e.forgetSpell(spell)
			}
			ok = false
		}
	}
	e.clearCombatText()
	if e.gameState == stateCombat {
		e.ClearCells(0, 0x17, 0x27, 0x17)
	}
}

// spellRemoveCurse = O0C2:3605 uncurse: the first target loses the curse
// affect, or else its first cursed item is put away; an item with a
// magic property takes it back and the scores are worked out again.
func (e *Engine) spellRemoveCurse() {
	t := e.tgt(1)
	if e.CureAffect(t, affectHampered) {
		e.hitMessage(t, true, "is un-cursed")
		return
	}
	found := false
	for _, it := range t.Items {
		if found {
			break
		}
		if it.Cursed == 0 {
			continue
		}
		found = true
		it.Readied = 0
		if it.Spells[2] > itemPropertyFlag {
			e.itemCall = true
			e.callItem = it
			e.callAffect(it.Spells[2], t, nil, 1)
			for k := statStr; k <= statCha; k++ {
				e.statAffects(t, k)
			}
		}
	}
	if found {
		e.hitMessage(t, true, "has an item un-cursed")
	}
}

// fxCursed = O0C2:371B curse.
func (e *Engine) fxCursed() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "has been cursed!") }

// fxBlinking = ovr23:263D.
func (e *Engine) fxBlinking() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is blinking") }

// fxFireball = ovr23:266B: a burst of radius 2 round the target square,
// level d6 of fire (spell 0x40 rolls its own level); the near list only
// becomes the target list when the area holds no word at +0x1CC.
func (e *Engine) fxFireball() {
	var lvl int
	if e.curSpell == 0x40 {
		lvl = int(byte(e.Dice(1, 3)*2 + 1))
	} else {
		lvl = e.casterLevel(int(e.curSpell))
	}
	if wordAt(e.area, 0x1CC) == 0 {
		e.buildNear(e.tgtX, e.tgtY, 2, 0xFF, 1)
		for i, n := range e.near.list {
			e.setTgt(i+1, e.combatant(n.index))
		}
		e.nSpellTargets = len(e.near.list)
	}
	e.combatCentre(e.tgtX, e.tgtY, 0, 8)
	e.o23_08C3(int(e.curSpell), 0, 0, e.RollDice(lvl, 6), 9, "")
}

// removeAffectFromSide = O0C2:387B (type, side, msg): the first casterLevel
// targets of that side lose the affect; the others are dropped. A nil
// target (garbage in the asm) is dropped as well.
func (e *Engine) removeAffectFromSide(typ, side int, msg string) {
	msg = cut(msg, 0x28)
	e.rv.areaSpell = 1
	lvl := byte(e.casterLevel(int(e.curSpell)))
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		if t != nil && int(t.Side) == side && lvl > 0 {
			lvl--
			if e.CureAffect(t, byte(typ)) {
				e.setTgt(i, nil)
			}
		} else {
			e.setTgt(i, nil)
		}
	}
	e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, msg)
}

// fxHaste = O0C2:395C cast_haste: slow is removed from the caster's side.
func (e *Engine) fxHaste() { e.removeAffectFromSide(0x2A, int(e.selected().Side), "is Hasted") }

// Ground kinds by groundInfo's first byte.
const (
	groundOpen    = 1   // a bolt flies on over it
	groundStopper = 255 // a wall: a bolt bounces off it, in a dungeon
)

// strikeSquare = O0C2:3986 (x, y, dmg, saveType, prevWho, var bounced):
// whoever stands on the square, unless it was the one struck last, takes
// dmg (half on a save, DS:6FAF = 12). bounced is set when the ground stops
// a bolt in a dungeon and it had not just bounced, else cleared.
func (e *Engine) strikeSquare(x, y, dmg, saveType int, prevWho byte, bounced *bool) {
	who, ground := e.mapAt(x, y)
	*bounced = ground > 0 && groundInfo[ground%64][0] == groundStopper &&
		wordAt(e.area, 2*areaWordInside) == 1 && !*bounced
	if who > 0 && who != prevWho {
		t := e.combatant(who)
		e.rv.dmgType = 0xC
		e.damagePerson(t, int(byte(dmg)), 2, e.SavingThrow(t, saveType, 0))
		e.hitColours(0x13)
		e.rv.dmgType = 0
	}
}

// boltPath = O0C2:3A44 (reach, dmg, saveType, bounce): the bolt goes on
// past the target square, away from the caster, for reach*2 steps, hitting
// each square; where it bounces it comes back the other way (once, when
// bounce is set and the caster is near, those squares count 8 more).
func (e *Engine) boltPath(reach, dmg, saveType, bounce int) {
	var bounced bool
	e.hitColours(0x13)
	prevWho, _ := e.mapAt(e.tgtX, e.tgtY)
	var used byte // v61b: the steps already taken off the reach
	dirSign := 1
	sel := e.selected()
	cx, cy := e.combatX(sel), e.combatY(sel)
	flag := bounce
	if cx == e.tgtX && cy == e.tgtY {
		return
	}
	r := byte(reach * 2)
	e.rv.areaSpell = 1
	for r > 0 {
		x0, y0 := e.tgtX, e.tgtY
		x1 := int(int16(e.tgtX + (e.tgtX-cx)*dirSign*int(r)))
		y1 := int(int16(e.tgtY + (e.tgtY-cy)*dirSign*int(r)))
		l := newLine(x0, y0, x1, y1)
		var who, ground byte
		for {
			lastX, lastY := byte(l.x), byte(l.y)
			if !(x0 == x1 && y0 == y1) {
				for {
					more := e.step(l)
					who, ground = e.mapAt(int(byte(l.x)), int(byte(l.y)))
					if groundInfo[ground%64][0] == groundOpen {
						bounced = false
					}
					if !more || who > 0 && who != prevWho || ground == 0 ||
						groundInfo[ground%64][0] > groundOpen || byte(l.steps) >= r {
						break
					}
				}
			}
			if ground == 0 {
				r = 0
			}
			e.flyMissile(int(lastX), int(lastY), int(byte(l.x)), int(byte(l.y)), 4, 0x32)
			e.strikeSquare(int(byte(l.x)), int(byte(l.y)), dmg, saveType, prevWho, &bounced)
			prevWho = who
			if bounced {
				e.tgtX, e.tgtY = int(byte(l.x)), int(byte(l.y))
				back := newLine(e.tgtX, e.tgtY, cx, cy)
				for e.step(back) {
				}
				if flag != 0 && byte(back.steps) <= 8 {
					l.steps += 8
				}
				dirSign = -dirSign
				flag = 0
				prevWho = 0
			}
			used = byte(l.steps) - used
			if used < r {
				r -= used
			} else {
				r = 0
			}
			used = byte(l.steps)
			if bounced || r == 0 {
				break
			}
		}
	}
	e.rv.areaSpell = 0
}

// fxLightning = O0C2:3CD9: level d6 down a line of 7, bouncing.
func (e *Engine) fxLightning() {
	d := byte(e.Dice(e.casterLevel(int(e.curSpell)), 6))
	var bounced bool // an unset local in 3CD9
	e.strikeSquare(e.tgtX, e.tgtY, int(d), 4, 0, &bounced)
	e.boltPath(7, int(d), 4, 1)
}

// fxSlow = O0C2:3D2E.
func (e *Engine) fxSlow() {
	e.removeAffectFromSide(0x27, b2i(isParty(e.selected())), "is Slowed")
}

// Restore's search: classes up to level 13, below 10,000,000 points.
const (
	maxRestoredLevel  = 13
	restoreExpCeiling = 10000000
)

// fxRestore = O0C2:3D68 cast_restore: gives back one drained level (the class that
// needs the fewest experience points) and the points lost with it.
func (e *Engine) fxRestore() {
	t := e.tgt(1)
	if t.Drained == 0 {
		return
	}
	q := t.DrainedHP / t.Drained
	t.MaxHP += q
	t.HP += q
	t.HPRoll += q
	t.DrainedHP -= q
	t.Drained--
	best := int8(maxRestoredLevel)
	bestXP := int32(restoreExpCeiling)
	var bestClass int // uninitialised in the asm when nothing qualifies
	for k := 0; k <= 7; k++ {
		lvl := int8(t.Levels[k])
		if lvl <= 0 || lvl > best {
			continue
		}
		xp := expThreshold(k, int(lvl))
		if xp <= 0 || xp >= bestXP {
			continue
		}
		if atRaceLimitAfterRestore(t, k) {
			continue
		}
		best, bestClass, bestXP = lvl, k, xp
	}
	t.Levels[bestClass]++
	if int32(t.Exp) < bestXP {
		t.Exp = uint32(bestXP)
	}
	e.recalcClassBonuses(t)
	e.combatName(t, "is restored", 0xA, true)
}

// fxSpeedy = O0C2:3F2C cast_speed.
func (e *Engine) fxSpeedy() {
	if !e.CureAffect(e.tgt(1), affectSlow) {
		e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is Speedy")
	}
}

// fxCureSerious = O0C2:3F6D: heals 2d8+1.
func (e *Engine) fxCureSerious() {
	if e.nSpellTargets == 0 {
		return
	}
	t := e.tgt(1)
	if e.Heal(t, e.Dice(2, 8)+1, false) {
		e.healedMessage(t)
	}
}

// fxStronger = O0C2:3FB7 cast_strength: giant strength 21 for 1d4*10+40 rounds.
func (e *Engine) fxStronger() {
	t := e.tgt(1)
	out, ok := strongerThan(t, giantStrength21, 0)
	if ok {
		e.combatName(t, "is stronger", combatMsgRow, true)
	}
	e.AddAffect(t, affectGiantStrength, e.Dice(1, 4)*10+40, out, 1)
	e.statAffects(t, statStr)
}

// fxBolt = O0C2:403C: Dice(1,6)+0x14 on the target square, then a fixed
// 0x14 along the line.
func (e *Engine) fxBolt() {
	var bounced bool
	e.strikeSquare(e.tgtX, e.tgtY, e.Dice(1, 6)+0x14, 4, 0, &bounced)
	e.boltPath(3, 0x14, 4, 0)
}

// fxParalyzed = ovr23:2F99.
func (e *Engine) fxParalyzed() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is paralyzed") }

// fxHeal2d4 = O0C2:40C2 cast_heal.
func (e *Engine) fxHeal2d4() {
	t := e.tgt(1)
	if e.Heal(t, e.Dice(2, 4)+2, false) {
		e.hitMessage(t, true, "is Healed")
	}
}

// fxInvisible = ovr23:302A.
func (e *Engine) fxInvisible() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is invisible") }

// fxDamage2d4 = O0C2:414A: 2d4+2 of damage type 8.
func (e *Engine) fxDamage2d4() {
	e.o23_08C3(int(e.curSpell), 0, 0, e.RollDice(2, 4)+2, 8, "")
}

// fxDamage2d8 = O0C2:4185: 2d8+1 of damage type 8.
func (e *Engine) fxDamage2d8() {
	e.o23_08C3(int(e.curSpell), 0, 0, e.RollDice(2, 8)+1, 8, "")
}

// o23_30D9 = O0C2:5A5D cast_breath (char, affect, flag): the lightning
// breath (affect 88). After the first round, half the time nothing
// happens; the monster breathes at the target square with its hit points
// as damage.
func (e *Engine) o23_30D9(c *Character, a *Affect, flag int) {
	if e.combatRound != 0 && e.Dice(1, 100) <= 0x32 {
		return
	}
	e.rv.dmgType = dmgElectricity | dmgType32
	cx, cy := e.combatX(c), e.combatY(c)
	e.combatName(c, "Breathes!", 0xA, true)
	e.spellTargetHook(0x33, 1) // the result (DS:70C6) is not used
	e.tgtX = cx + sign(e.tgtX-cx)
	e.tgtY = cy + sign(e.tgtY-cy)
	if cx+1 == e.tgtX {
		e.tgtX++
	}
	if cy+1 == e.tgtY {
		e.tgtY++
	}
	e.removeInvisibility(c)
	e.hitColours(0x13)
	e.flyMissile(cx, cy, e.tgtX, e.tgtY, 4, 0x32)
	var bounced bool
	e.strikeSquare(e.tgtX, e.tgtY, int(c.MaxHP), 3, 0, &bounced)
	e.boltPath(0xA, int(c.MaxHP), 3, 0)
	if a.Data > 0xFD {
		a.Data--
	} else {
		e.RemoveAffect(c, affectBreathLightning, a)
	}
	e.endTurn(c)
}

// scrollUsed = ovr23:328A (char, item, spell): the spell is wiped from the
// scroll; the scroll goes when the item byte +0x30 falls under 0xD2.
func (e *Engine) scrollUsed(c *Character, it *Item, spell byte) {
	s := spell & 0x7F
	idx := 0
	for i := 1; i <= 3; i++ {
		if it.Spells[i-1]&0x7F == s {
			idx = i
		}
	}
	if idx == 0 {
		return
	}
	it.Spells[idx-1] = 0
	it.Words[1]--
	if it.Words[1] < 0xD2 {
		e.RemoveItem(c, it)
	}
}

// castMessage = O0C2:648D cast_a_spell (char, msg, spell): announces the
// cast, in the side panel in combat, else in the message box.
func (e *Engine) castMessage(c *Character, msg string, spell int) {
	msg = cut(msg, 0x28)
	if e.gameState == stateCombat {
		e.combatName(c, "Casts a Spell", combatMsgRow, true)
		e.ClearCells(0, 0x17, 0x27, 0x17)
		e.DisplayString(0, 0x17, colourGreen, "Spell:"+spellNames[spell])
		return
	}
	e.ClearCells(1, 0x12, 0x26, 0x16)
	e.displayName(c, 1, 0x13, false)
	e.DisplayString(len(c.Name())+2, 0x13, colourGreen, msg)
	e.DisplayString(1, 0x14, colourGreen, spellNames[spell])
	e.MsgDelay()
	e.clearCombatText()
}

// spellTableInit = ovr23:3413: the DS:6B84 table of spell effects (indexed
// by spell, slot 0 is spellTargetSetup), the breath affect handler (type
// 0x58, DS:6988) and the spell globals.
func (e *Engine) spellTableInit() {
	e.rv.cure = 0 // DS:681F
	e.usingItem = false
	e.spellCaster = nil
	e.itemsRedraw = true // DS:6821
	h := &affectHandlers
	h[affectStoningGaze] = (*Engine).stoningGazeHandler
	h[affectSpitAcid] = (*Engine).spitAcidHandler
	h[affectBreathLightning] = func(e *Engine, c *Character, a *Affect, flag byte) { e.o23_30D9(c, a, int(flag)) }
	h[affectBreathAcid] = (*Engine).breathAcidHandler
	h[affectGazeParalyse] = (*Engine).gazeParalyseHandler
	h[affectBreathFire] = (*Engine).breathFireHandler
	h[affectBreathFireClose] = (*Engine).breathFireCloseHandler
	h[affectThrowLightning] = (*Engine).throwLightningHandler
	e.spellFx[1] = e.o23_100F
	e.spellFx[2] = e.o23_1040
	e.spellFx[3] = e.o23_106B
	e.spellFx[4] = e.o23_10A7
	e.spellFx[5] = e.o23_10E9
	e.spellFx[6] = e.o23_1123
	e.spellFx[7] = e.o23_1123
	e.spellFx[8] = e.o23_1162
	e.spellFx[9] = e.o23_1190
	e.spellFx[10] = e.o23_11DD
	e.spellFx[11] = e.o23_10E9
	e.spellFx[12] = e.spellStronger
	e.spellFx[13] = e.spellReduce
	e.spellFx[14] = e.spellFriends
	e.spellFx[15] = e.o23_1439
	e.spellFx[16] = e.o23_1123
	e.spellFx[17] = e.o23_1123
	e.spellFx[18] = e.o23_10E9
	e.spellFx[19] = e.o23_1499
	e.spellFx[20] = e.o23_14C7
	e.spellFx[21] = e.o23_151B
	e.spellFx[22] = e.o23_10E9
	e.spellFx[23] = e.o23_1658
	e.spellFx[24] = e.o23_17E9
	e.spellFx[25] = e.o23_1822
	e.spellFx[26] = e.o23_185B
	e.spellFx[27] = e.o23_190F
	e.spellFx[28] = e.o23_19C3
	e.spellFx[29] = e.o23_10E9
	e.spellFx[30] = e.o23_1A16
	e.spellFx[31] = e.o23_1A4F
	e.spellFx[32] = e.o23_1A8A
	e.spellFx[33] = e.o23_1ACC
	e.spellFx[34] = e.o23_1B11
	e.spellFx[35] = e.spellStrength
	e.spellFx[36] = e.o23_206F
	e.spellFx[37] = e.o23_2217
	e.spellFx[38] = e.o23_225D
	e.spellFx[39] = e.o23_232F
	e.spellFx[40] = e.o23_234C
	e.spellFx[41] = e.o23_2385
	e.spellFx[42] = e.o23_24D6
	e.spellFx[43] = e.spellRemoveCurse
	e.spellFx[44] = e.fxCursed
	e.spellFx[45] = e.fxBlinking
	e.spellFx[46] = e.o23_2385
	e.spellFx[47] = e.fxFireball
	e.spellFx[48] = e.fxHaste
	e.spellFx[49] = e.o23_1658
	e.spellFx[50] = e.o23_1A16
	e.spellFx[51] = e.fxLightning
	e.spellFx[52] = e.o23_1123
	e.spellFx[53] = e.o23_1123
	e.spellFx[54] = e.o23_1123
	e.spellFx[55] = e.fxSlow
	e.spellFx[56] = e.fxRestore
	e.spellFx[57] = e.fxSpeedy
	e.spellFx[58] = e.fxCureSerious
	e.spellFx[59] = e.fxStronger
	e.spellFx[60] = e.fxBolt
	e.spellFx[61] = e.fxParalyzed
	e.spellFx[62] = e.fxHeal2d4
	e.spellFx[63] = e.fxInvisible
	e.spellFx[64] = e.fxFireball
	e.spellFx[65] = e.fxDamage2d4
	e.spellFx[66] = e.fxDamage2d8
	e.spellFx[67] = e.fxNeutralizePoison
	e.spellFx[68] = e.fxPoison
	e.spellFx[69] = e.o23_1123
	e.spellFx[70] = e.fxSticksToSnakes
	e.spellFx[71] = e.fxCureCritical
	e.spellFx[72] = e.fxCauseCritical
	e.spellFx[73] = e.fxDispelEvil
	e.spellFx[74] = e.fxFlameStrike
	e.spellFx[75] = e.fxRaiseDead
	e.spellFx[76] = e.fxSlayLiving
	e.spellFx[77] = e.o23_10E9
	e.spellFx[78] = e.fxEntangle
	e.spellFx[79] = e.fxFaerieFire
	e.spellFx[80] = e.fxInvisibleToAnimals
	e.spellFx[81] = e.fxCharmMonsters
	e.spellFx[82] = e.fxConfusion
	e.spellFx[83] = e.fxDimensionDoor
	e.spellFx[84] = e.fxFear
	e.spellFx[85] = e.fxFireShield
	e.spellFx[86] = e.fxFumble
	e.spellFx[87] = e.fxIceStorm
	e.spellFx[88] = e.fxMinorGlobe
	e.spellFx[89] = e.spellRemoveCurse
	e.spellFx[90] = e.o23_206F
	e.spellFx[91] = e.fxCloudkill
	e.spellFx[92] = e.fxConeOfCold
	e.spellFx[93] = e.fxFeeblemind
	e.spellFx[94] = e.o23_1658
	e.spellFx[95] = e.fxPlain
	e.spellFx[96] = e.fxPlain
	e.spellFx[97] = e.fxPlain
	e.spellFx[98] = e.fxSpell98
	e.spellFx[99] = e.fxHeal2d4
	e.spellFx[100] = e.fxCursed
}

// o23_37B0 = ovr23:37B0: empty (entry 0).
func (e *Engine) o23_37B0() {}
