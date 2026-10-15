package engine

// Spell effect routines of ovr23 (0x100F..0x2540). CastSpell (ovr23:0C20)
// calls them through the table DS:6B84 filled by ovr23:3413, indexed by the
// spell number DS:681B; each is called without arguments and works on the
// spell globals: spellID (DS:681B), spellN/spellList (DS:6C94/6C91, list
// entry 1 is DS:6C95, the first target).


// o23_100F = ovr23:100F: Bless, "is Blessed" on the selected character's side.
func (e *Engine) o23_100F() {
	e.o23_0F4F(int(e.selected().Side), "is Blessed")
}

// o23_1040 = ovr23:1040: Bane, "is Cursed".
func (e *Engine) o23_1040() {
	side := 0
	if isParty(e.selected()) {
		side = 1
	}
	e.o23_0F4F(side, "is Cursed")
}

// o23_106B = ovr23:106B: Cure Light Wounds on the first target, 1d8.
func (e *Engine) o23_106B() {
	if e.nSpellTargets != 0 {
		if e.Heal(e.tgt(1), e.Dice(1, 8), false) {
			e.healedMessage(e.tgt(1))
		}
	}
}

// o23_10A7 = ovr23:10A7: 1d8 magic damage.
func (e *Engine) o23_10A7() {
	e.o23_08C3(int(e.curSpell), 0, 0, e.RollDice(1, 8), 8, "")
}

// o23_10E9 = ovr23:10E9: "is affected" (shared by several spells).
func (e *Engine) o23_10E9() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is affected") }

// o23_1123 = ovr23:1123: "is protected".
func (e *Engine) o23_1123() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is protected") }

// o23_1162 = ovr23:1162: "is cold-resistant".
func (e *Engine) o23_1162() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is cold-resistant") }

// o23_1190 = ovr23:1190: damage equal to the caster level, type 9.
func (e *Engine) o23_1190() {
	e.o23_08C3(int(e.curSpell), 0, 0, e.casterLevel(int(e.curSpell)), 9, "")
}

// o23_11DD = ovr23:11DD: Charm Person; large or strong targets are unaffected.
func (e *Engine) o23_11DD() {
	t := e.tgt(1)
	if t.Kind > 1 || t.Size > 1 {
		e.combatName(t, "is unaffected", 0xA, true)
		return
	}
	lvl := e.casterLevel(int(e.curSpell))
	e.o23_08C3(int(e.curSpell), (int(e.selected().Side)<<7+lvl)&0xFF, 1, 0, 0, "is charmed")
	if a := e.FindAffect(t, 0xB); a != nil {
		e.callAffect(0xB, e.tgt(1), a, 0)
	}
}

// spellStronger = O0C2:2005 is_stronger: Strength. The strength comes from
// the caster level: 18 with 18/00..18/100 for levels 1..6, then 19..22.
func (e *Engine) spellStronger() {
	t := e.spellTarget
	str, str00 := byte(strengthMaximum18), byte(0)
	switch e.casterLevel(int(e.curSpell)) {
	case 2:
		str00 = 1
	case 3:
		str00 = 51
	case 4:
		str00 = 76
	case 5:
		str00 = 91
	case 6:
		str00 = 100
	case 7:
		str = 19
	case 8:
		str = 20
	case 9:
		str = 21
	case 10, 11:
		str = 22
	}
	if code, ok := strongerThan(t, str, str00); ok {
		e.combatName(t, "is stronger", combatMsgRow, true)
		e.AddAffect(t, affectEnlarge, e.o23_07D6(int(e.curSpell)), code, 1)
		e.statAffects(t, statStr)
		return
	}
	e.combatName(t, "is unaffected", combatMsgRow, true)
}

// spellReduce = O0C2:213A has_been_reduced: the magical strength (affect
// 12) goes unless the target saves.
func (e *Engine) spellReduce() {
	t := e.spellTarget
	if t != nil && e.nSpellTargets > 0 && !e.SavingThrow(t, saveVsSpell, 0) && e.FindAffect(t, affectEnlarge) != nil {
		e.RemoveAffect(t, affectEnlarge, nil)
		e.statAffects(t, statStr)
		e.combatName(t, "has been reduced", combatMsgRow, true)
	}
}

// spellFriends = O0C2:21DA is_friendly: the charisma affect at 2d4.
func (e *Engine) spellFriends() {
	e.o23_08C3(int(e.curSpell), e.Dice(2, 4), 1, 0, 0, "is friendly")
	e.statAffects(e.selected(), statCha)
}

// o23_1439 = O0C2:2221: magic damage of ((level+1)/2)d4 + (level+1)/2.
func (e *Engine) o23_1439() {
	n := (e.casterLevel(int(e.curSpell)) + 1) & 0xFF >> 1
	e.o23_08C3(int(e.curSpell), 0, 0, (n+e.RollDice(n, 4))&0xFF, 8, "")
}

// o23_1499 = ovr23:1499: "is shielded".
func (e *Engine) o23_1499() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is shielded") }

// o23_14C7 = ovr23:14C7: caster level + 1d8 damage, type 0xC.
func (e *Engine) o23_14C7() {
	lvl := e.casterLevel(int(e.curSpell))
	e.o23_08C3(int(e.curSpell), 0, 0, (lvl+e.RollDice(1, 8))&0xFF, 0xC, "")
}

// o23_151B = ovr23:151B: Sleep. 4d4 hit dice of targets fall asleep, weakest
// first in list order; the others (and the already asleep) leave the list.
func (e *Engine) o23_151B() {
	e.rv.areaSpell = 1
	e.v4844 = byte(e.Dice(4, 4))
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		switch lvl := int8(t.MaxLevel); {
		case lvl >= 0 && lvl <= 1:
			e.v4845 = 1
		case lvl == 2:
			e.v4845 = 2
		case lvl == 3:
			e.v4845 = 4
		case lvl == 4:
			e.v4845 = 6
		case lvl == 5:
			if t.Race == 0 {
				e.v4845 = 0xA
			} else {
				e.v4845 = 0x14
			}
		default:
			e.v4845 = 0x14
		}
		if t.Status != 1 && e.FindAffect(t, 0x35) == nil && e.v4844 >= e.v4845 {
			e.v4844 -= e.v4845
		} else {
			e.setTgt(i, nil)
		}
	}
	e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "falls asleep")
}

// Spells with their own rules in holdTargets.
const (
	spellHoldPerson  = 23
	spellHoldAutoA   = 79 // the last target never saves
	spellHoldAutoB   = 81
	spellHoldAnySize = 94 // large and strong targets get no free save
)

// o23_1658 = O0C2:2444 is_held: Hold Person/Monster. The save bonus
// depends on the number of targets.
func (e *Engine) o23_1658() {
	var bonus int8 // other counts leave the local unset (0)
	switch e.nSpellTargets {
	case 1:
		if e.curSpell == spellHoldPerson {
			bonus = -2
		} else {
			bonus = -3
		}
	case 2:
		bonus = -1
	}
	e.holdTargets(bonus, "is held")
}

// holdTargets = O0C2:1B24 (bonus, msg): each target, last first, gets a
// missile (but the first one shot) and the spell record's affect unless it
// saves.
func (e *Engine) holdTargets(bonus int8, msg string) {
	spell := int(e.curSpell)
	for i := e.nSpellTargets; i >= 1; i-- {
		t := e.tgt(i)
		if t == nil {
			continue
		}
		if i < e.nSpellTargets {
			e.PlaySound(sndSpell)
			e.hitColours(0x12)
			sel := e.selected()
			e.flyMissile(e.combatX(sel), e.combatY(sel), e.combatX(t), e.combatY(t), 4, 0x1E)
		}
		var saved bool
		var saveMode byte
		if (spell == spellHoldAutoA || spell == spellHoldAutoB) && i == e.nSpellTargets {
			saved, saveMode = true, 1
		} else {
			saved = e.SavingThrow(t, int(e.spellRec(byte(spell), 9)), int(bonus))
			saveMode = e.spellRec(byte(spell), 8)
		}
		if (t.Kind > 1 || t.Size > 1) && spell != spellHoldAnySize {
			saved = true
		}
		e.isUnaffected(t, e.spellRec(byte(spell), 10), e.o23_07D6(spell), byte(e.casterLevel(spell)), 0,
			saveMode, saved, msg)
	}
}

// o23_17E9 = ovr23:17E9: "is fire resistant".
func (e *Engine) o23_17E9() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is fire resistant") }

// o23_1822 = ovr23:1822: "is silenced".
func (e *Engine) o23_1822() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is silenced") }

// o23_185B = ovr23:185B: Raise Dead style: a dead target is dropped from
// the list; one with affect 0x37 is brought to 1 hit point and stunned (0x0F).
func (e *Engine) o23_185B() {
	t := e.tgt(1)
	if t.Status == 1 {
		e.setTgt(1, nil)
		return
	}
	if e.FindAffect(t, 0x37) == nil {
		return
	}
	if t.HP == 0 {
		t.HP = 1
	}
	e.o23_08C3(int(e.curSpell), 0xFF, 1, 0, 0, "is affected")
	e.callAffect(0x4E, t, nil, 1)
	e.AddAffect(t, 0xF, 10, 0xFF, 1)
}

// o23_190F = ovr23:190F: the party members of kind 0xE whose hit points
// fit in the caster's hit points become the targets.
func (e *Engine) o23_190F() {
	e.v4844 = e.selected().HP
	e.nSpellTargets = 0
	for _, p := range e.party {
		if p.Kind == 0xE && e.v4844 >= p.HP {
			e.v4844 -= p.HP
			e.nSpellTargets++
			e.setTgt(e.nSpellTargets, p)
		}
	}
	e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is charmed")
}

// o23_19C3 = ovr23:19C3: applies the spell, then runs the affect 0x17 handler on the first target.
func (e *Engine) o23_19C3() {
	e.o23_08C3(int(e.curSpell), 0, 1, 0, 0, "")
	e.callAffect(0x17, e.tgt(1), nil, 0)
}

// o23_1A16 = ovr23:1A16: "is invisible".
func (e *Engine) o23_1A16() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is invisible") }

// o23_1A4F = ovr23:1A4F: Knock.
func (e *Engine) o23_1A4F() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "Knock-Knock") }

// o23_1A8A = O0C2:2748 is_duplicated: Mirror Image, 1d4 images in the high
// nibble of the level.
func (e *Engine) o23_1A8A() {
	lvl := byte(e.Dice(1, 4))<<4 | byte(e.casterLevel(int(e.curSpell)))
	e.o23_08C3(int(e.curSpell), int(lvl), 0, 0, 0, "is duplicated")
}

// o23_1ACC = ovr23:1ACC: "is weakened".
func (e *Engine) o23_1ACC() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is weakened") }

// o23_1B11 = Stinking Cloud (O0C2 create_noxious_cloud).
func (e *Engine) o23_1B11() { e.createCloud(e.stinkingKind()) }

// hasClass: a level in the class now, or an old one above the level of
// the class taken up after (a dual-classed character's first class).
func hasClass(c *Character, k int) bool {
	return int8(c.Levels[k]) > 0 || int8(c.OldLevels[k]) > int8(c.MulticlassLevel)
}

// spellStrength = O0C2:2C5B: Strength. 1d4 for a mage, 1d6 for a cleric or
// thief, 1d8 for a fighter (the last that applies) on the strength in use;
// above 18 the fighters, paladins and rangers get exceptional strength.
func (e *Engine) spellStrength() {
	var str00, add byte // add is an unset local when no class qualifies
	t := e.tgt(1)
	if hasClass(t, clsMage) {
		add = byte(e.Dice(1, 4))
	}
	if hasClass(t, clsCleric) || hasClass(t, clsThief) {
		add = byte(e.Dice(1, 6))
	}
	if hasClass(t, clsFighter) {
		add = byte(e.Dice(1, 8))
	}
	str := t.Stats[statStr].Value + add
	if str > strengthMaximum18 {
		if hasClass(t, clsFighter) || hasClass(t, clsPaladin) || hasClass(t, clsRanger) {
			str00 = t.Str00.Value + (str-strengthMaximum18)*10
			if str00 > 100 {
				str00 = 100
			}
		}
		str = strengthMaximum18
	}
	if _, ok := strongerThan(t, str, str00); ok {
		e.AddAffect(t, affectStrengthSpell, e.o23_07D6(int(e.curSpell)), add+strengthEncodedBase, 1)
		e.statAffects(t, statStr)
	}
}

// o23_206F = O0C2:2E19 is_animated: Animate Dead. Corpses of the party list (status 6,
// kind 0) are put back on their feet on the caster's side, one per caster level.
func (e *Engine) o23_206F() {
	e.rv.areaSpell = 1
	lvl := e.casterLevel(int(e.curSpell))
	e.nSpellTargets = 0
	sel := e.selected()
	for _, p := range e.party {
		if lvl <= 0 {
			break
		}
		if p.Status != 6 || p.Kind != 0 || !e.combatSquareFree(p, e.combatX(p), e.combatY(p), true) {
			continue
		}
		data := byte(int(p.Side)<<4 + e.casterLevel(int(e.curSpell)))
		p.Side = sel.Side
		p.Auto = 1
		p.Undead = 2
		p.FightLevel = 0
		p.BaseMove = 6
		p.Memorized = [len(p.Memorized)]byte{}
		if p.Control > 0x7F {
			p.Control = 0xB2
		} else {
			p.Control = 0xB3
		}
		p.Kind = 4
		if e.gameState == 5 {
			p.Combat.Target = nil
		}
		lvl--
		if e.Revive(p, p.MaxHP) {
			e.isUnaffected(p, 0x20, 0, data, 1, 0, false, "is animated")
			p.Status = 1
		}
	}
}

// o23_2217 = ovr23:2217: cures blindness (affect 0x21): "can see".
func (e *Engine) o23_2217() {
	if e.CureAffect(e.tgt(1), 0x21) {
		e.hitMessage(e.tgt(1), true, "can see")
	}
}

// o23_225D = ovr23:225D: "is blind".
func (e *Engine) o23_225D() { e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is blind") }

// cureDiseaseAffects = ovr23:228A: with the cure flag (DS:681F) set, cures
// affects 0x22, 0x2B (and 0x2C, 0x1F) and 0x32 (and 0x39) of the first target.
func (e *Engine) cureDiseaseAffects() bool {
	cured := false
	e.rv.cure = 1
	t := e.tgt(1)
	if e.CureAffect(t, 0x22) {
		cured = true
	}
	if e.CureAffect(e.tgt(1), 0x2B) {
		cured = true
		e.RemoveAffect(e.tgt(1), 0x2C, nil)
		e.RemoveAffect(e.tgt(1), 0x1F, nil)
	}
	if e.CureAffect(e.tgt(1), 0x32) {
		cured = true
		e.RemoveAffect(e.tgt(1), 0x39, nil)
	}
	e.rv.cure = 0
	return cured
}

// o23_232F = ovr23:232F: Cure Blindness spell body (the result is unused).
func (e *Engine) o23_232F() { e.cureDiseaseAffects() }

// o23_234C = ovr23:234C: "is diseased".
func (e *Engine) o23_234C() { e.o23_08C3(int(e.curSpell), 0, 1, 0, 0, "is diseased") }

// Dispel Magic's odds: 50%, +5% for each caster level above the affect's,
// -2% for each below.
const (
	dispelOdds      = 50
	dispelPerAbove  = 5
	dispelPerBelow  = 2
	affectLevelMask = 0x0F // an affect's data: the caster level it was cast at
)

// dispelChance: the odds of a caster level against an affect's level.
func dispelChance(lvl, against byte) byte {
	switch {
	case lvl > against:
		return (lvl-against)*dispelPerAbove + dispelOdds
	case lvl < against:
		return dispelOdds - (against-lvl)*dispelPerBelow
	}
	return dispelOdds
}

// o23_2385 = O0C2:31E4 is_affected3: Dispel Magic. Each pass walks the
// affects of the first target (the loop variable only counts passes);
// each temporary affect may go. Then the clouds on the nine squares round
// the target square: a cloud whose caster's mage level does not stop it
// lifts, else it stays for good.
func (e *Engine) o23_2385() {
	e.rv.areaSpell = 1
	lvl := byte(e.casterLevel(int(e.curSpell)))
	for i := 1; i <= e.nSpellTargets; i++ {
		hit := false
		t := e.tgt(1)
		for _, a := range append([]*Affect(nil), t.Affects...) {
			if a.Data == 0xFF {
				continue
			}
			e.v4845 = a.Data & affectLevelMask
			e.v4844 = dispelChance(lvl, e.v4845)
			if byte(e.Dice(1, 100)) <= e.v4844 {
				e.RemoveAffect(t, a.Type, a)
				hit = true
			}
		}
		if hit {
			e.hitMessage(t, true, "is affected")
		}
	}
	var x, y int8
	for k := 0; k <= 8; k++ {
		tx, ty := int8(e.tgtX), int8(e.tgtY)
		switch k {
		case 0:
			x, y = tx, ty
		case 1, 8:
			y = ty - 1
		case 2:
			x = tx + 1
		case 3, 7:
			y = ty
		case 4:
			y = ty + 1
		case 5:
			x = tx
		case 6:
			x = tx - 1
		}
		_, g := e.mapAt(int(x), int(y))
		var list []*cloudRecord
		var n int
		var affect byte
		switch g {
		case groundPoisonCloud:
			list, n, affect = e.poisonClouds, 9, affectCloudkill
		case groundStinkingCloud:
			list, n, affect = e.stinkingClouds, 4, affectStinkingCloud
		default:
			continue
		}
		for _, r := range append([]*cloudRecord(nil), list...) {
			for j := 0; j < n; j++ {
				if sx, sy := r.square(dispelCloudSquares[:], j); sx != int(x) || sy != int(y) || r.stays {
					continue
				}
				if !e.dispelsCloud(r.caster, lvl) {
					r.stays = true
					continue
				}
				for _, a := range r.caster.Affects {
					if a.Type == affect && a.Data>>4 == r.number {
						e.RemoveAffect(r.caster, affect, a)
						break
					}
				}
			}
		}
	}
}

// dispelsCloud = O0C2:3126 (caster): the dispel roll against the mage
// level of a cloud's caster (old levels counted by the selected
// character's dual class).
func (e *Engine) dispelsCloud(caster *Character, lvl byte) bool {
	old := byte(0)
	if oldClassUsable(e.selected()) {
		old = 1
	}
	e.v4845 = caster.OldLevels[clsMage]*old + caster.Levels[clsMage]
	e.v4844 = dispelChance(lvl, e.v4845)
	return byte(e.Dice(1, 100)) <= e.v4844
}

// o23_24D6 = ovr23:24D6: "is praying", cast at caster level with the side in the high nibble.
func (e *Engine) o23_24D6() {
	e.o23_08C3(int(e.curSpell), int(e.selected().Side)<<4+e.casterLevel(int(e.curSpell)), 0, 0, 0, "is praying")
}

// fxNeutralizePoison = O0C2:41DA cure_poison: an animated target drops
// out; a poisoned one (affect 55) is back on its feet with at least a hit
// point, without its poison affects.
func (e *Engine) fxNeutralizePoison() {
	t := e.tgt(1)
	if t.Status == statusAnimated {
		e.setTgt(1, nil)
		return
	}
	if e.FindAffect(t, affectPoisoned) != nil {
		if t.HP == 0 {
			t.HP = 1
		}
		e.rv.cure = 1
		e.RemoveAffect(t, affectPoisoned, nil)
		e.RemoveAffect(t, affectSlowPoison, nil)
		e.RemoveAffect(t, affectPoisonDamage, nil)
		e.rv.cure = 0
		e.combatName(t, "is unpoisoned", combatMsgRow, true)
		t.Okay = 1
		t.Status = statusOkay
		return
	}
	e.combatName(t, "is unaffected", combatMsgRow, true)
}

// fxPoison = O0C2:42D0: the spell's affect on the targets; then the
// caster's own target is tried against the poison sting (affect 64): when
// nothing there blocks it, the sting handler runs for the caster.
func (e *Engine) fxPoison() {
	e.o23_08C3(int(e.curSpell), 0, 0, 0, dmgMagic, "")
	sel := e.selected()
	e.rv.affVal = affectPoisonSting
	e.affectGroup(sel.Combat.Target, 9)
	if e.rv.affVal == affectPoisonSting {
		e.callAffect(affectPoisonSting, sel, nil, 0) // the affect argument is an unset local
	}
}

// sticksToSnakesTooBig: the hit dice from which a target is too big for the snakes.
const sticksToSnakesTooBig = 6

// fxSticksToSnakes = O0C2:435A cast_flattern: a small target gets the
// snakes (and their handler runs at once); a big one smashes them flat.
func (e *Engine) fxSticksToSnakes() {
	t := e.tgt(1)
	if int8(t.MaxLevel) < sticksToSnakesTooBig {
		e.o23_08C3(int(e.curSpell), e.casterLevel(int(e.curSpell)), 0, 0, dmgMagic, "")
		if a := e.FindAffect(t, affectSticksToSnakes); a != nil {
			e.callAffect(affectSticksToSnakes, t, a, 0)
		}
		return
	}
	e.combatName(t, "smashes them flat", combatMsgRow, true)
}

// fxCureCritical = O0C2:43F0: heals 3d8+3.
func (e *Engine) fxCureCritical() {
	if e.nSpellTargets == 0 {
		return
	}
	t := e.tgt(1)
	if e.Heal(t, e.Dice(3, 8)+3, false) {
		e.healedMessage(t)
	}
}

// fxCauseCritical = O0C2:4431: 3d8+3 of damage type 8.
func (e *Engine) fxCauseCritical() {
	e.o23_08C3(int(e.curSpell), 0, 0, e.RollDice(3, 8)+3, dmgMagic, "")
}

// fxDispelEvil = O0C2:4479 is_affected4: the caster is warded against
// enchanted creatures (affect 4, Dispel Evil's own amount), then the spell's
// affect on the targets.
func (e *Engine) fxDispelEvil() {
	e.isUnaffected(e.selected(), affectWardEnchanted, e.o23_07D6(spellDispelEvil), 0, 0, 0, false, "")
	e.o23_08C3(int(e.curSpell), 0, 0, 0, 0, "is affected")
}

// fxFlameStrike = O0C2:44DA: 6d8 of damage type 9.
func (e *Engine) fxFlameStrike() {
	e.o23_08C3(int(e.curSpell), 0, 0, e.RollDice(6, 8), dmgMagic|dmgFire, "")
}

// fxRaiseDead = O0C2:451A cast_raise: a dead (or animated) target with
// constitution left and not an elf comes back with a hit point, a point of
// constitution less.
func (e *Engine) fxRaiseDead() {
	t := e.tgt(1)
	if (t.Status == statusDead || t.Status == statusAnimated) && t.Stats[statCon].Base > 0 && t.Race != raceElf {
		e.rv.cure = 1
		e.RemoveAffect(t, affectCollapse, nil)
		e.RemoveAffect(t, affectPoisoned, nil)
		e.rv.cure = 0
		t.Status = statusOkay
		t.Okay = 1
		t.Stats[statCon].Base--
		e.statAffects(t, statCon)
		t.HP = 1
		e.combatName(t, "is raised", combatMsgRow, true)
	}
}
