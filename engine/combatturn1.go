package engine

// Combat turn, first half (ovr14:0000..1BF8): setting up a combatant's turn,
// movement, attacks of opportunity, fleeing, sweeps, turning undead and the
// attack itself. The second half (from 1BF8) is ported separately.

// turnSetup = ovr14:0000 (char): fill the combat block for a new round.
func (e *Engine) o14_0000(c *Character) {
	cb := c.Combat
	cb.Spell, cb.CanCast, cb.CanUse, cb.Attacked, cb.AttackSlot = 0, 1, 1, 0, 2
	e.o14_0D23(c)
	e.v681A = c.BaseAttack[1]
	e.v681D = 0
	e.affectGroup(c, grpSpeed)
	c.Attacks[1] = byte(e.o14_0E52(e.v681A))
	cb.Sweeps = c.FightLevel
	if c.Okay != 0 {
		cb.Delay = byte(int(dexMissile(c)) + e.Dice(1, 6))
		if int8(cb.Delay) < 1 {
			cb.Delay = 1
		}
		if byte((int(int8(c.Side))+1)&int(wordAt(e.game, area2Surprise))) != 0 { // only the low byte is tested
			cb.Delay -= 6
		}
		if int8(cb.Delay) < 0 || int8(cb.Delay) > 0x14 {
			cb.Delay = 0
		}
	} else {
		cb.Delay = 0
	}
	cb.Moves = byte(e.o14_0123(c))
}

// o14_0123 (char): movement rate of the turn (half of it is kept in +0x113
// by o14_0D23); the affects of group 0x12 may change it.
func (e *Engine) o14_0123(c *Character) int {
	v := c.Move
	if c.Side == 0 {
		v += byte(wordAt(e.game, area2PartyMoveBonus))
	}
	e.v681D = 1
	if v < 1 || v > 0x60 {
		v = 1
	}
	e.v681A = v << 1
	e.affectGroup(c, grpSpeed)
	e.v681D = 0
	return int(e.v681A)
}

// o14_0191 (att, vic, slot): damage roll of the attack slot (1 or 2),
// tripled-or-so for a backstab, then the
// affects of groups 4 (attacker) and 5 (victim) may change DS:6818.
func (e *Engine) o14_0191(a, b *Character, slot int) {
	d := int(int8(a.Bonus[slot-1])) + e.RollDice(int(a.Dice[slot-1]), int(a.Sides[slot-1]))
	e.rv.damage = byte(d)
	// (the original compares the byte with 0: never clamps)
	if e.canBackstab(a, b) {
		usable := 0
		if oldClassUsable(a) {
			usable = 1
		}
		lvl := int(int8(a.Levels[clsThief])) + int(int8(a.OldLevels[clsThief]))*usable
		e.rv.damage = byte(int(e.rv.damage) * (int(uint16(int16(lvl-1)))>>2 + 2))
	}
	e.rv.dmgType = 0
	e.affectGroup(a, grpDamageBonus)
	e.affectGroup(b, grpWeaponDamage)
}

// Attack text kinds.
const (
	attackPlain      = 0
	attackFromBehind = 1
	attackBackstab   = 2
	attackSlay       = 3
)

// Side panel cells of the attack text.
const (
	attackTextColumn = 23
	attackTextRow    = 12
	attackTextRight  = 38
	attackTextLines  = 3
)

// o14_02F9 = O08D:0318 backstab (att, vic, kind, damage, shown, hit): the
// attack text. kind: attackPlain .. attackSlay; damage is applied, shown
// printed; hit 0 is a miss.
func (e *Engine) o14_02F9(a, b *Character, kind, damage, shown, hit int) {
	var s string
	switch kind {
	case attackBackstab:
		s = "-Backstabs-"
	case attackSlay:
		s = "slays helpless"
	default:
		s = "Attacks"
	}
	e.combatName(a, s, combatMsgRow, false)
	row := attackTextRow
	e.displayName(b, attackTextColumn, row, false)
	row++
	s = ""
	if kind == attackFromBehind {
		s = "(from behind) "
	}
	if hit != 0 {
		if kind == attackSlay {
			s = "with one cruel blow"
		} else {
			s += "Hitting for " + itoa(shown&0xFF)
			if shown&0xFF == 1 {
				s += " point "
			} else {
				s += " points "
			}
			s += "of damage"
		}
		e.damagePlayer(b, byte(damage))
	} else {
		s += "and Misses"
	}
	if b.Status != statusDisintegrated {
		e.DisplayTextBlock(attackTextColumn, row, attackTextRight, row+attackTextLines, colourGreen, false, s)
	}
	row = int(e.msgRow) + 1
	if damage&0xFF > 0 {
		b.Combat.CanCast = 0
		if b.Combat.Spell > 0 {
			e.gameDelay()
			e.combatName(b, "lost a spell", combatMsgRow+2, true)
			forgetSpellOf(b, b.Combat.Spell)
			b.Combat.Spell = 0
		} else {
			e.gameDelay()
		}
	} else {
		e.gameDelay()
	}
	if b.Okay == 0 {
		e.combatName(b, "goes down", row, false)
		row += 2
		if b.Status == statusDying {
			e.DisplayString(attackTextColumn, row, colourGreen, "and is Dying")
		}
		if st := b.Status; st >= statusDead && st <= statusDisintegrated {
			e.combatName(b, "is killed", row, false)
		}
		e.downAffects(b)
		e.affectGroup(b, grpDown)
		if b.Okay == 0 {
			e.combatRemoveFromMap(b)
		} else {
			e.gameDelay()
		}
	}
	e.clearCombatText()
}

// o14_0627 (char): the enemies next to a mover that were waiting (+7) lose
// their reaction, turn to it and attack it.
func (e *Engine) o14_0627(c *Character) {
	n := e.nearEnemies(c, 1)
	for i := 1; i <= n; i++ {
		if c.Okay == 0 {
			continue
		}
		t := e.combatant(e.targets[i])
		if t.Combat.Guarding == 0 || e.isHelpless(t) {
			continue
		}
		e.combatCentre(e.combatX(c), e.combatY(c), 2, 8)
		t.Combat.Guarding = 0
		e.o14_17F6(t, c)
		e.o14_1884(t, c, 0, nil)
	}
}

// o14_0712 = O08D:0748 (char, dir): step one square, paying the ground's
// step cost (stepCost) out of the moves left.
func (e *Engine) o14_0712(c *Character, dir int) {
	cb := c.Combat
	idx := e.combatIndex(c)
	ox, oy := e.cpos[idx].x, e.cpos[idx].y
	nx := ox + byte(stepX[dir])
	ny := oy + byte(stepY[dir])
	cost := stepCost(e.groundInfo(e.cmap.ground[ny][nx])[0], dir)
	if cost > int(cb.Moves) {
		cb.Moves = 0
	} else {
		cb.Moves -= byte(cost)
	}
	margin := 1
	if c.Auto != 0 {
		margin = 3
		if !e.combatInView(int(int8(nx))-e.cmap.viewX, int(int8(ny))-e.cmap.viewY) && e.showMoves {
			e.combatCentre(int(ox), int(oy), 2, 8)
		}
	}
	if e.showMoves {
		e.combatClearIcon(0, 0, idx)
	}
	e.cpos[idx].x, e.cpos[idx].y = nx, ny
	e.combatRedrawMap()
	if e.showMoves {
		e.combatCentre(int(nx), int(ny), margin, 8)
	}
	cb.Attackers, cb.Flank = 0, 0
	e.PlaySound(sndStep)
	e.o14_0627(c)
	if c.Okay == 0 || e.isHelpless(c) {
		cb.Moves = 0
	}
}

// o14_08B2 (dir, char): the mover steps (virtually) one square in dir; the
// enemies that were next to it and no longer are get a free attack.
func (e *Engine) o14_08B2(c *Character, dir int) {
	var lst [0x0D]byte // bp-0xD+i, i = 1..: the DS:6DE4 list before the move
	idx := e.combatIndex(c)
	n := e.nearEnemies(c, 1)
	if n == 0 {
		return
	}
	for i := 1; i <= n; i++ {
		lst[i] = e.targets[i]
	}
	dx, dy := byte(stepX[dir]), byte(stepY[dir])
	e.cpos[idx].x += dx
	e.cpos[idx].y += dy
	m := e.nearEnemies(c, 1)
	e.cpos[idx].x -= dx
	e.cpos[idx].y -= dy
	for i := 1; i <= n; i++ {
		found := false
		for j := 1; j <= m; j++ {
			if e.targets[j] == lst[i] {
				found = true
			}
		}
		if found {
			lst[i] = 0
		}
	}
	for i := 1; i <= n; i++ {
		if lst[i] == 0 || c.Okay == 0 {
			continue
		}
		e.panelDirty, e.showMoves = true, true
		t := e.combatant(lst[i])
		if e.isHelpless(t) || !e.o14_1085(t, c) {
			continue
		}
		// (affect 75 is the bonus against kind 3 and 74 is unused: Pool's
		// held and holding numbers, kept by CotAB)
		if e.FindAffect(t, affectVsKind3) != nil || e.FindAffect(t, 74) != nil {
			continue
		}
		attacked := false
		tcb := t.Combat
		for k := int(tcb.Facing) + 6; k <= int(tcb.Facing)+10; k++ {
			if attacked {
				continue
			}
			if int8(tcb.Delay) <= 0 && tcb.Attackers != 0 {
				if !e.inArc(e.combatX(t), e.combatY(t), e.combatX(c), e.combatY(c), k%8) {
					continue
				}
			}
			w := 1
			if t.BaseAttack[0] == 0 {
				w = 2
			}
			for s := 1; s <= 2; s++ {
				if t.Attacks[s-1] > 0 {
					w = s
				}
			}
			if t.Attacks[w-1] == 0 {
				t.Attacks[w-1] = 1
			}
			tcb.AttackSlot = byte(w)
			saved := t.Combat.Target
			e.o14_1884(t, c, 1, nil)
			attacked = true
			t.Combat.Target = saved
			if c.Okay != 0 {
				e.panelDirty = true
				e.combatPanel(c)
			}
		}
	}
}

// o14_0C6A (char) bool: try to run away; "Got Away" removes the character.
func (e *Engine) o14_0C6A(c *Character) bool {
	away := false
	if e.nearEnemies(c, 0xFF) == 0 {
		away = true
	} else {
		a := byte(e.o14_0123(c)) >> 1
		b := byte(e.o14_2908(c))
		if b < a {
			away = true
		} else if b == a && e.Dice(1, 2) == 1 {
			away = true
		}
	}
	if away {
		e.KillCharacter(c, 3, "Got Away")
	} else {
		e.BottomMessage("Escape is blocked")
	}
	return e.endTurn(c)
}

// o14_0D23 (char): the character's speed in +0x113 from its movement base
// (+0xA1) or the readied weapon's speed (DS:5582 record +5, at least 2).
func (e *Engine) o14_0D23(c *Character) {
	flag := false
	old := c.Attacks[0]
	c.Attacks[0] = c.BaseAttack[0]
	var it *Item
	if e.rangedWeapon(c) && e.ammoFor(c, &it) {
		flag = true
		v := byte(0) // (the original reads the item at +0xCC unchecked)
		if c.Slots[0] != nil {
			v = e.baseItem(c.Slots[0].Type, 5)
		}
		if v < 2 {
			v = 2
		}
		e.v681A = v
	} else {
		e.v681A = c.Attacks[0]
	}
	e.v681D = 0
	e.affectGroup(c, 0x12)
	r := byte(e.o14_0E52(e.v681A))
	if flag && it != nil {
		w := byte(1)
		if it.Count > w {
			w = it.Count
		}
		if w < r {
			r = w
		}
	}
	cb := c.Combat
	if cb.Attacked != 0 && r >= old && (int(r) >= int(old)*2 || flag) {
		return
	}
	c.Attacks[0] = r
}

// o14_0E52 (v): half of v, rounded up when DS:6DE3 is odd.
func (e *Engine) o14_0E52(v byte) int {
	if e.combatRound&1 != 0 {
		v++
	}
	return int(v >> 1)
}

// o14_0E82 (att, vic) bool: a fighter with more attacks than... sweeps: the
// attacker (fewer attacks than +5 allows, victim not a "big" target) hits
// several adjacent weak enemies, one attack each.
func (e *Engine) o14_0E82(a, b *Character) bool {
	res := false
	acb := a.Combat
	if a.Attacks[0] < acb.Sweeps && b.MaxLevel == 0 && e.rangeTo(a, b) == 1 {
		n := e.nearEnemies(a, 1)
		cnt := 0
		bi := 0
		for i := 1; i <= n; i++ {
			t := e.combatant(e.targets[i])
			if t == b {
				bi = i
			}
			if t.MaxLevel == 0 {
				cnt++
			}
		}
		if byte(cnt) > a.Attacks[0] {
			if byte(cnt) > acb.Sweeps {
				cnt = int(acb.Sweeps)
			}
			e.CharMessage(a, "sweeps", 0xA, true)
			if e.combatant(e.targets[1]) != b {
				e.targets[bi] = e.targets[1] // (bi 0 when b was not found: DS:6DE4)
				e.targets[1] = byte(e.combatIndex(b))
			}
			for i := 1; i <= n; i++ {
				if cnt > 0 {
					t := e.combatant(e.targets[i])
					if t.MaxLevel == 0 {
						e.o14_17F6(a, t)
						a.Attacks[0] = 1
						e.o14_1884(a, t, 0, nil)
						cnt--
					}
				}
			}
			res = true
		}
	}
	return res
}

// o14_1085 (t, c) bool: may t act against c? Affect group 1 of c can veto
// (DS:681E); then group 0 of t is run with c as its target.
func (e *Engine) o14_1085(t, c *Character) bool {
	if c == nil {
		return false
	}
	if t == c {
		return true
	}
	e.v681E = 0
	e.affectGroup(c, grpAttackerExtra)
	if e.v681E == 0 {
		saved := t.Combat.Target
		t.Combat.Target = c
		e.affectGroup(t, grpNone)
		t.Combat.Target = saved
	}
	return e.v681E == 0
}

// o14_116A (char): a cleric turns undead (ovr14:116A).
func (e *Engine) o14_116A(c *Character) {
	e.combatName(c, "turns undead...", combatMsgRow, false)
	e.ClearRow24() // O104 redraw_screen
	e.gameDelay()
	any, stop := false, false
	c.Combat.TurnUsed = 1
	limit := 6
	hd := e.Dice(1, 0xC)
	roll := e.Dice(1, 0x14)
	var cls int
	usable := 0
	if oldClassUsable(c) {
		usable = 1
	}
	switch lv := int8(int(c.Levels[clsCleric]) + int(c.OldLevels[clsCleric])*usable); {
	case lv >= 1 && lv <= 8:
		cls = int(c.Levels[clsCleric]) // the original's column is the current level alone
	case lv >= 9 && lv <= 13:
		cls = 9
	default:
		cls = 10
	}
	var t *Character
	for e.o14_1351(c, &t) && hd > 0 && !stop {
		x := int(turnUndeadRolls[int(t.Undead)-1][cls-1])
		ax := x
		if ax < 0 {
			ax = -ax
		}
		if roll < ax {
			stop = true
			continue
		}
		any = true
		e.combatRedrawChar(t, 3, 0)
		e.panelDirty = true
		e.combatPanel(t)
		if x > 0 {
			t.Combat.Turned = 1
			e.hitMessage(t, true, "is turned")
		} else {
			e.combatName(t, "Is destroyed", combatMsgRow, false)
			e.combatRemoveFromMap(t)
			t.Status = statusDisintegrated
			t.Okay = 0
		}
		if limit > 0 {
			limit--
		}
		hd--
		if hd == 0 && limit > 0 && x < 0 {
			hd++
		}
		e.clearCombatText()
	}
	if !any {
		e.printMessage("Nothing Happens...")
	}
	e.countSides()
	e.endTurn(c)
	e.clearCombatText()
}

// o14_1351 (var out, char) bool: the weakest (lowest +0x76 > 0) enemy not
// yet turned, within reach.
func (e *Engine) o14_1351(c *Character, out **Character) bool {
	best := int8(0xD)
	*out = nil
	n := e.nearEnemies(c, 0xFF)
	found := false
	for i := 1; i <= n; i++ {
		t := e.combatant(e.targets[i])
		if t.Combat.Turned == 0 && int8(t.Undead) > 0 && int8(t.Undead) < best {
			best = int8(t.Undead)
			*out = t
			found = true
		}
	}
	return found
}

// o14_1406 (att, vic, w, var done): the attacker's attack slots against the
// victim; done is 1 when the attacker has used all its attacks.
func (e *Engine) o14_1406(a, b *Character, w int, it *Item) (done bool) {
	acb := a.Combat
	done = false
	e.hits6822[1], e.hits6822[2] = 0, 0
	e.shots6E2C[1], e.shots6E2C[2] = 0, 0
	any, stop := false, false
	e.rv.damage = 0
	acb.Attacked = 1
	if e.isHelpless(b) {
		e.PlaySound(sndBump)
		for acb.AttackSlot > 0 && a.Attacks[acb.AttackSlot-1] == 0 {
			acb.AttackSlot--
		}
		e.shots6E2C[acb.AttackSlot]++
		e.o14_02F9(a, b, attackSlay, (int(b.HP)+5)&0xFF, 1, 1)
		e.removeInvisibility(a)
		for s := 1; s <= 2; s++ {
			a.Attacks[s-1] = 0
		}
		any = true
		done = true
	} else {
		if a.Slots[0] != nil && (b.Size > 0x80 || b.Size&7 > 1) {
			typ := a.Slots[0].Type
			a.Dice[0] = e.baseItem(typ, 2)
			a.Sides[0] = e.baseItem(typ, 3)
			a.Bonus[0] = byte(int(int8(a.Bonus[0])) - int(int8(e.baseItem(typ, 0xB))) + int(int8(e.baseItem(typ, 4))))
		}
		e.ApplyItems(b)
		e.affectGroup(b, grpArmour)
		hand := 0
		var need byte
		if e.canBackstab(a, b) {
			need = b.AC[1] - 4
		} else {
			if b.Combat.Attackers > 1 && byte(e.o14_263D(a, b)) == b.Combat.Facing && b.Combat.Flank > 4 {
				w = 1
			}
			if w != 0 {
				hand = 1
			}
			need = b.AC[hand]
		}
		bonus := int(need)
		e.o14_1B1E(a, b, &bonus)
		need = byte(bonus)
		mode := attackPlain
		if w != 0 {
			mode = attackFromBehind
		}
		if e.canBackstab(a, b) {
			mode = attackBackstab
		}
		for s := int(acb.AttackSlot); s >= 1; s-- {
			for a.Attacks[s-1] != 0 && !stop {
				a.Attacks[s-1]--
				acb.AttackSlot = byte(s)
				e.shots6E2C[s]++
				if e.AttackHits(a, b, int(need)) || e.isHelpless(b) {
					e.hits6822[s]++
					e.PlaySound(sndBump)
					any = true
					e.o14_0191(a, b, s)
					e.o14_02F9(a, b, mode, int(e.rv.damage), int(e.rv.damage), 1)
					if b.Okay != 0 {
						e.affectGroup(a, s+1)
					}
					if b.Okay == 0 {
						stop = true
					}
				}
			}
		}
		if it != nil && it.Count == 0 && it.Type == itemTypeThrownOnce {
			a.Attacks[0], a.Attacks[1] = 0, 0
		}
		if !any {
			e.PlaySound(sndMiss)
			e.o14_02F9(a, b, mode, 0, 0, 0)
		}
		done = true
		for s := 1; s <= 2; s++ {
			if a.Attacks[s-1] > 0 {
				done = false
			}
		}
		acb.Sweeps = 0
	}
	if a.Okay == 0 {
		done = true
	}
	if done {
		e.endTurn(a)
	}
	return
}

// Weapons that animate their attack, and the flag of a thrown weapon that
// comes back.
const (
	itemTypeLightning  = 47
	itemTypeShootsBolt = 101
	itemReturns        = 137
)

// itemTypeThrownOnce: a thrown weapon of this type ends the attacks when
// the last one is gone.
const itemTypeThrownOnce = 100

// o14_17F6 (att, vic): the victim counts one more attack and turns towards
// the attacker (the turned amount accumulates in +0x12).
func (e *Engine) o14_17F6(att, vic *Character) {
	cb := vic.Combat
	cb.Attackers++
	d := (e.o14_263D(vic, att) - int(cb.Facing) + 8) % 8
	if d > 4 {
		d = 8 - d
	}
	cb.Flank = byte((int(cb.Flank) + d) % 8)
}

// o14_1884 (att, vic, flag, item, var done): a full attack of att on vic
// (turning both, then the slots of o14_1406). item is a thrown item that is
// spent; done is set when the attacker's turn ends.
func (e *Engine) o14_1884(a, b *Character, flag int, it *Item) (done bool) {
	e.showMoves = true
	e.panelDirty = true
	// (O08D:19DB also writes round + 15 to the first byte of the near list)
	var dir byte
	if b.Combat.Attackers < 2 && flag == 0 {
		dir = byte(e.o14_263D(b, a))
		b.Combat.Facing = (dir + 4) % 8
	} else if e.combatOnScreen(b, false) {
		dir = b.Combat.Facing
		if flag == 0 {
			b.Combat.Facing = (dir + 4) % 8
		}
	}
	if e.combatOnScreen(b, false) {
		e.combatTurn(b, dir, 0, false)
	}
	dir = byte(e.o14_263D(a, b))
	e.combatPanel(a)
	e.combatTurn(a, dir, 1, false)
	a.Combat.Target = b
	e.Delay(0x64)
	if it != nil {
		e.attackAnimation(a, b, it)
	}
	if w := a.Slots[0]; w != nil && (w.Type == itemTypeLightning || w.Type == itemTypeShootsBolt) {
		e.attackAnimation(a, b, w)
	}
	done = true
	if a.Attacks[0] > 0 || a.Attacks[1] > 0 {
		saved := e.partySel
		for i, c := range e.party {
			if c == a {
				e.partySel = i
			}
		}
		done = e.o14_1406(a, b, flag, it)
		if it != nil {
			if it.Count > 0 {
				it.Count -= e.shots6E2C[1]
			}
			if it.Count == 0 {
				if e.throwable(a) && it.Spells[2] != itemReturns {
					cp := *it
					n := &cp
					e.run.treasure = append([]*Item{n}, e.run.treasure...)
					n.Readied = 0
					e.loseItem(a, it)
					e.v5D9A = n
				} else {
					e.loseItem(a, it)
				}
			}
		}
		e.ApplyItems(a)
		e.partySel = saved
	}
	if done {
		done = e.endTurn(a)
	}
	if e.combatOnScreen(a, false) {
		e.combatTurn(a, a.Combat.Facing, 1, true)
		e.combatTurn(a, a.Combat.Facing, 0, false)
	}
	return
}

// o14_1B1E (att, vic, var need): the attacker's range band: every full
// weapon range band beyond the first adds 2, then 3, to the number to hit.
func (e *Engine) o14_1B1E(a, b *Character, need *int) {
	d := e.rangeTo(a, b)
	f := d
	if e.rangedWeapon(a) {
		f = (int(e.baseItem(a.Slots[0].Type, 0xC)) - 1) / 3
	}
	if d-f > 0 {
		d -= f
		*need += 2
	}
	if d-f > 0 {
		d -= f
		*need += 3
	}
}
