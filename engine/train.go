package engine

import (
	"cotab/dax"
	"cotab/ega"
)

// ovr17 (second half): Modify, Train and Add on the party screen, the
// portrait and icon editors, and the hit point helpers they share.

// partyExp = es:[game+0x550]: the classes this training hall teaches
// (a mask that is ANDed with DS:5DE; the party menu only offers Train
// when it is set).
func (e *Engine) partyExp() int { return int(wordAt(e.game, 0x550)) }

// modifyDraw = ovr17:2033 (idx, highlight): the ability line idx of the
// selected character, or its hit points for idx 6.
func (e *Engine) modifyDraw(idx int, hi bool) {
	if idx < 6 {
		e.showStat(idx, hi) // ovr20:0901
	} else {
		e.displayHP(e.selected(), 4, 0x12, hi) // 00BA:0973
	}
}

// ModifyCharacter = ovr17:209E: change the abilities and hit points of a
// character that has no experience yet, within the race and class limits
// (DS:3DA3.., DS:3F81..). Left/right change the line, up/down move,
// Keep (K) accepts, Exit (E) puts the old values back.
func (e *Engine) ModifyCharacter() {
	c := e.selected()
	if int32(c.Exp) > 0 {
		e.DisplayMessageWait(0xE, c.Name()+" can't be modified.")
		return
	}
	e.CharacterSheet() // ovr20:004A
	saved := c.Stats
	savedStr00, savedHP := c.Str00, c.MaxHP
	tbl := func(off int) byte { return e.X.Bytes(off, 1)[0] }
	race, class, sex := int(c.Race), int(c.Class), int(c.Sex)
	minOff := [6]int{0x3DA3 + sex, 0x3DA9, 0x3DAB, 0x3DAD, 0x3DAF, 0x3DB1}
	maxOff := [6]int{0x3DA5 + sex, 0x3DAA, 0x3DAC, 0x3DAE, 0x3DB0, 0x3DB2}
	idx := 6
	e.modifyDraw(idx, false)
	idx = 0
	e.modifyDraw(idx, true)
	// hpToDraw redraws the hit point line, leaving the cursor on idx.
	hpToDraw := func() {
		c.HP = c.MaxHP
		save := idx
		idx = 6
		e.modifyDraw(idx, false)
		idx = save
	}
	// slots: when wisdom changes, a first level cleric slot is capped at 1.
	wisdom := func() {
		if c.SpellSlots[0] > 0 {
			c.SpellSlots[0] = 1
		}
		e.wisdomSlots(c, false) // ovr24:01AF
	}
	lower := func(i int) {
		c.Stats[i].Cur--
		if i == 0 && c.Str00.Cur > 0 {
			c.Str00.Cur--
			c.Stats[statStr].Cur++
		} else if lo := tbl(minOff[i] + race*16); c.Stats[i].Cur < lo || c.Stats[i].Cur < tbl(0x3F81+class*6+i) {
			// Quirk: below the class minimum the race minimum is stored.
			c.Stats[i].Cur = lo
		}
		switch i {
		case 2:
			wisdom()
		case 4:
			if m := e.maxHitPoints(c); m < c.MaxHP {
				c.MaxHP = m
			}
			hpToDraw()
		}
	}
	raise := func(i int) {
		c.Stats[i].Cur++
		if hi := tbl(maxOff[i] + race*16); c.Stats[i].Cur > hi {
			if i == 0 && c.Stats[statStr].Cur > 0x12 && int8(c.Levels[clsFighter]) > 0 && tbl(0x3DA7+race*16+sex) > c.Str00.Cur {
				c.Str00.Cur++ // exceptional strength for fighters
			}
			c.Stats[i].Cur = hi
		}
		switch i {
		case 2:
			wisdom()
		case 4:
			if m := e.minHitPoints(c); m > c.MaxHP {
				c.MaxHP = m
			}
			hpToDraw()
		}
	}
	defer e.rawArrows("↑↓=score|← →=lower, raise|K, click=keep|Esc=cancel")()
	for {
		key, extra := e.DisplayInput("Modify character: ", "Keep Exit", 0xD, 0xA, 0xF, true, false)
		e.modifyDraw(idx, false)
		if !extra && (key == 'E' || key == 0) {
			c.Stats = saved
			c.Str00, c.MaxHP = savedStr00, savedHP
			c.HP = c.MaxHP
			e.ApplyItems(c) // 00BA:0BB8
			return
		}
		if extra {
			switch key {
			case 0x50, 0x0D:
				if idx++; idx > 6 {
					idx = 0
				}
			case 0x48:
				if idx--; idx < 0 {
					idx = 6
				}
			case 0x4B:
				if idx < 6 {
					lower(idx)
				} else {
					c.MaxHP--
					if m := e.minHitPoints(c); m > c.MaxHP {
						c.MaxHP = m
					}
					c.HP = c.MaxHP
				}
			case 0x4D:
				if idx < 6 {
					raise(idx)
				} else {
					c.MaxHP++
					if m := e.maxHitPoints(c); m < c.MaxHP {
						c.MaxHP = m
					}
					c.HP = c.MaxHP
				}
			}
		}
		e.ApplyItems(c)
		e.showCombatStats() // ovr20:0608
		e.modifyDraw(idx, true)
		if !extra && key == 'K' {
			c.Share = 1
			return
		}
	}
}

// hitPointsFrom = the tail of ovr17:3FB3/408C: sum plus the constitution
// bonus, averaged over the classes (count 0 would be a divide error in
// the original; 0 here).
func (e *Engine) hitPointsFrom(c *Character, sum byte, count int) byte {
	if count == 0 {
		return 0
	}
	adj := e.conHitPointBonusOf(c)
	if adj < 0 {
		if int(sum) > count+(-adj) {
			return byte((int(sum) + adj) / count)
		}
		return 1
	}
	return byte((int(sum) + adj) / count)
}

// minHitPoints = ovr17:3FB3 (char): the least hit points the levels allow.
func (e *Engine) minHitPoints(c *Character) byte {
	var sum byte
	count := 0
	for i := 0; i <= 7; i++ {
		if lv := int8(c.Levels[i]); lv > 0 {
			sum += byte(lv)
			count++
			if i == 4 || i == 7 {
				sum++
			}
		}
	}
	return e.hitPointsFrom(c, sum, count)
}

// maxHitPoints = ovr17:408C (char): the most hit points the levels allow
// (a full hit die per level, the die of each class).
func (e *Engine) maxHitPoints(c *Character) byte {
	var sum byte
	count := 0
	for i := 0; i <= 7; i++ {
		lv := int(int8(c.Levels[i]))
		if lv <= 0 {
			continue
		}
		count++
		switch i {
		case 0, 1:
			sum += byte(lv << 3)
		case 2, 3:
			sum += byte(lv * 10)
		case 4:
			sum += byte((lv + 1) << 3)
		case 5:
			sum += byte(lv << 2)
		case 6:
			sum += byte(lv * 6)
		case 7:
			sum += byte((lv + 1) << 2)
		}
	}
	return e.hitPointsFrom(e.selected(), sum, count) // the original reads the bonus of the selected character, not c
}

// AddCharacter = ovr17:306D: pick saved characters from the roster list
// and add them to the party (at most 6 characters and 8 with hired NPCs).
// Chosen entries get a "* " in the list.
func (e *Engine) AddCharacter() {
	e.ClearCells(1, 1, 0x26, 0x16)
	names := e.BuildCharList() // ovr18:0010
	if len(names) == 0 {
		return
	}
	list := make([]*ListNode, len(names))
	for i, n := range names {
		list[i] = &ListNode{Text: n}
	}
	index := 0
	redraw := true
	var sel *ListNode
	count := 0 // uninitialised until the first add in the original
	for {
		// The show-exit argument is an uninitialised local; false here.
		key := e.ListMenu("Add a character: ", "Add Exit", 0xD, 0xA, 0xF, 1, 2, 0x26, 0x16, list, false, &redraw, &index, &sel)
		if (key == 0xD || key == 'A') && sel != nil && (sel.Text == "" || sel.Text[0] != '*') {
			c := &Character{}
			if e.LoadCharacter(sel.Text, c, false) { // ovr18:0AEC
				sel.Text = cut("* "+sel.Text, 0x28)
				count = 0
				if len(e.party) == 0 {
					putWord(e.game, gsPartyCount, 0)
					e.AddToParty(c) // ovr18:1446
					e.BuildCharIcon(c, true)
				} else {
					var dup *Character
					for _, m := range e.party {
						if m.Name() == c.Name() && m.RandomID == c.RandomID {
							dup = m
							break
						}
						if m.Control < 0x80 {
							count++
						}
					}
					npc := c.Control >= 0x80
					if dup == nil && (!npc && count < 6 || npc && wordAt(e.game, gsPartyCount) < 8) {
						e.AddToParty(c)
						e.BuildCharIcon(c, true)
						if !npc {
							count++
						}
					}
				}
			}
		}
		if key == 'E' || key == 0 || count > 5 || wordAt(e.game, gsPartyCount) > 7 {
			break
		}
	}
	for i, n := range list {
		names[i] = n.Text
	}
	e.WriteCharList(names) // ovr18:01B8
}

// o17_33E7 = ovr17:33E7: choose the portrait: Head (1..14) and Body
// (1..12) step through the pictures, Keep accepts. There is no way out
// without Keep (Esc just asks again).
func (e *Engine) o17_33E7() {
	c := e.selected()
	head, body := c.PortHead, c.PortBody
	for {
		e.SheetPortrait() // ovr29:04F9
		key, extra := e.DisplayInput("", "Head Body Keep", 0, 0xF, 0xA, false, false)
		if !extra {
			switch key {
			case 'H':
				if c.PortHead < 0xE {
					c.PortHead++
				} else {
					c.PortHead = 1
				}
			case 'B':
				if c.PortBody < 0xC {
					c.PortBody++
				} else {
					c.PortBody = 1
				}
			case 'K':
				head, body = c.PortHead, c.PortBody
			}
		}
		if key == 'K' {
			break
		}
	}
	c.PortHead, c.PortBody = head, body
	e.ClearRow24()
}

// iconPane = ovr17:34DF (x, y) nested in the icon editor: the two poses
// of the icon in slot 0xC at icon square (x,y), on a cleared 12 cell
// panel (on EGA over the slot 0x19 background icons).
func (e *Engine) iconPane(x, y int) {
	if e.Cfg.Graphics != 'E' && e.Cfg.Graphics != 'T' { // DS:4A1E == 0 (1 EGA, 2 Tandy)
		e.FillRect(x*3, y*0x18, 0xC, 0x18, 3)
	} else {
		e.FillRect(x*3, y*0x18, 0xC, 0x18, 0)
		e.drawIcon(x, y, 0, 0, 0x19)
		e.drawIcon(x+3, y, 0, 1, 0x19)
	}
	e.drawIcon(x, y, 0, 0, 0xC)
	e.drawIcon(x+3, y, 0, 1, 0xC)
	e.Flip() // 0464:1B52
}

// iconCopy = ovr17:35BE (src, dst, recolour) nested in the icon editor:
// copy both poses of icon slot src into slot dst (the original also
// copies the mask planes, which the port keeps in the image); with
// recolour the selected character's six colour pairs (+0xC1..) replace
// the DS:3CF5 colours in the copy.
func (e *Engine) iconCopy(src, dst byte, recolour bool) {
	for p := 0; p < 2; p++ {
		if im := e.icons[src][p]; im != nil {
			e.icons[dst][p] = im.Clone()
		}
	}
	if !recolour {
		return
	}
	c := e.selected()
	var to [16]byte
	for i := range to {
		to[i] = byte(i)
	}
	pos := e.X.Bytes(0x3CF5, 7)
	for i := 1; i <= 6; i++ {
		v := c.IconColour[i-1]
		to[pos[i]] = v & 0xF
		to[pos[i]+8] = v >> 4
	}
	for p := 0; p < 2; p++ {
		recolourIcon(e.icons[dst][p], to)
	}
}

// recolourIcon = 0464:1E5C (identity, to, 0, 0, icon): colour i becomes
// to[i] for i = 0..15 in turn, so a pixel already moved to a later colour
// moves again (3 -> 15 -> to[15]). Mask pixels are left alone. Amiga
// pixels (extra) take the same end result through amigaIconColour.
func recolourIcon(im *dax.Image, to [16]byte) {
	if im == nil {
		return
	}
	var f, am [16]byte
	for c := range f {
		x := byte(c)
		for i := x; i < 16; i++ {
			if x == i {
				x = to[i]
			}
		}
		f[c] = x
		am[amigaIconColour[c]] = amigaIconColour[x]
	}
	for k, v := range im.Data {
		switch {
		case v < 16 && int(v) != im.Mask:
			im.Data[k] = f[v]
		case v >= ega.AmigaBase:
			im.Data[k] = v&^0xF | am[v&0xF]
		}
	}
}

// amigaIconColour (extra): the Amiga palette slot for each EGA colour
// number a character's icon colours hold. The Amiga icon editor (its slot
// table at 0x8B10E, against DS:3CF5) colours the parts in slots 9/1,
// 10/2, 11/3, 6/7, 5/8 and 15/14 where DOS uses 1/9 .. 7/15; the four
// left over pair by shade. So the DOS default colours give the Amiga
// default icon and each step shows another Amiga colour.
var amigaIconColour = [16]byte{0, 9, 10, 11, 6, 4, 5, 15, 12, 1, 2, 3, 7, 13, 8, 14}

// o17_382D = ovr17:382D: the combat icon editor (called from the
// character creation and the camp). States (menus DS:7BF + 0x29*state):
// 1 Parts/color-1/color-2/Size/Exit, 2 Head/Weapon, 3 the part to colour,
// 4 Size, 5 the Next/Prev/Keep/Exit stepping for the choice made before.
// The old icon is shown above the new one; Is this icon ok? ends it.
func (e *Engine) o17_382D() {
	c := e.selected()
	e.DrawBoxTitled(1, 1, 0x26, 0x16, 0, 0, "")
	e.combatViewLeave() // ovr32:00E3
	menu3 := e.X.PString(0x83A)
	var state, parent, part int // parent and part are uninitialised until set
	var sub, key byte
	var face bool
	for {
		e.BuildCharIcon(c, false) // ovr18:0482
		state = 1
		var colours [6]byte
		copy(colours[:], c.IconColour[:])
		order := c.Order
		c.Order = 0xC // the working copy is built into slot 0xC
		e.BuildCharIcon(c, false)
		c.Order = order
		head, weapon, size := c.IconHead, c.IconBody, c.IconSize
		e.iconCopy(order, 0xC, true)
		e.iconPane(1, 2)
		e.DisplayString(8, 6, 0xF, "old")
		e.DisplayString(3, 0xA, 0xF, "ready   action")
		e.DisplayString(8, 0xC, 0xF, "new")
		e.DisplayString(3, 0x10, 0xF, "ready   action")
		for {
			e.iconPane(1, 4)
			sizeStr := ""
			if state == 4 {
				sizeStr = "Large"
				if c.IconSize == 2 {
					sizeStr = "Small" // as in the original
				}
			}
			m := e.X.PString(0x7BF + state*0x29)
			if state == 3 {
				m = menu3
			}
			k, extra := e.DisplayInput("", sizeStr+m, 0xD, 0xA, 0xF, false, false)
			key = k
			if !extra {
				done := key == 0 || key == 'E'
				switch state {
				case 1:
					parent = 1
					switch key {
					case 'P':
						state = 2
					case '1', '2':
						state, face = 3, key == '2'
						label := "Hair"
						if face {
							label = "Face"
						}
						// Delete(s, 13, 4); Insert(label, s, 13) on DS:83A.
						s := menu3[:12] + menu3[16:]
						menu3 = cut(s[:12]+label+s[12:], 0x28)
					case 'S':
						state = 4
					}
				case 2:
					parent = 2
					if done {
						state = 1
					} else {
						sub, state = key, 5
					}
				case 3:
					parent = 3
					part = 1
					switch key {
					case 'W':
						part = 6
					case 'B':
						part = 1
					case 'H', 'F':
						part = 4
					case 'S':
						part = 5
					case 'A':
						part = 2
					case 'L':
						part = 3
					}
					if done {
						state = 1
					} else {
						state = 5
					}
				case 4:
					switch {
					case key == 'L':
						c.IconSize = 2
						e.BuildCharIcon(c, false)
					case key == 'S':
						c.IconSize = 1
						e.BuildCharIcon(c, false)
					case key == 'K':
						size, state, key = c.IconSize, 1, ' '
					case done:
						c.IconSize = size
						state, key = 1, ' '
					}
					e.BuildCharIcon(c, false)
				case 5:
					switch parent {
					case 2:
						// Step head (0..13) or weapon (0..31) pictures.
						pic, saved, top := &c.IconHead, &head, byte(0xD)
						if sub == 'W' {
							pic, saved, top = &c.IconBody, &weapon, 0x1F
						}
						if sub == 'H' || sub == 'W' {
							switch {
							case key == 'P':
								if *pic > 0 {
									*pic--
								} else {
									*pic = top
								}
							case key == 'N':
								if *pic < top {
									*pic++
								} else {
									*pic = 0
								}
							case key == 'K':
								*saved, state, key = *pic, parent, ' '
							case done:
								*pic = *saved
								state, key = parent, ' '
							}
							e.BuildCharIcon(c, false)
						}
					case 3:
						lo, hi := c.IconColour[part-1]&0xF, c.IconColour[part-1]>>4
						switch {
						case key == 'N', key == 'P':
							d := byte(1)
							if key == 'P' {
								d = 0xF // (n-1) and 15
							}
							if face {
								hi = (hi + d) & 0xF
							} else {
								lo = (lo + d) & 0xF
							}
							c.IconColour[part-1] = hi<<4 + lo
						case key == 'K':
							copy(colours[:], c.IconColour[:])
							state, key = parent, ' '
						case done:
							copy(c.IconColour[:], colours[:])
							state, key = parent, ' '
						}
					}
				}
			}
			e.iconCopy(c.Order, 0xC, true)
			if parent == 1 && (key == 0 || key == 'E') {
				break
			}
		}
		c.IconHead, c.IconBody, c.IconSize = head, weapon, size
		copy(c.IconColour[:], colours[:])
		e.iconCopy(c.Order, 0xC, true)
		e.iconCopy(0xC, c.Order, false)
		e.ClearRow24()
		e.freeIcon(0xC) // ovr33:014E
		if e.viewMode > 1 {
			e.LoadIconPic("ICON", 0, 0xB)
			e.LoadIconPic("ICON", 1, 0xC)
		}
		if e.YesNo(0xD, 0xA, 0xF, "Is this icon ok? ") == 'Y' {
			e.Music(3) // Amiga 0x2F33E
			break
		}
	}
	e.combatViewEnter() // ovr32:0105
}

// Experience bits of the classes, for the trainer's mask and the mask of
// classes being trained.
func classBit(class int) byte { return classExpMasks[class] }

// expThreshold is the experience a class needs to reach the level after
// level (0 when there is none).
func expThreshold(class, level int) int32 {
	row := expTableData[class][:]
	i := (level + 1) * 4
	if i+4 > len(row) { // the table rows run into the next class (99-byte rows)
		next := expTableData[min(class+1, len(expTableData)-1)][:]
		row = append(append([]byte{}, row...), next...)
	}
	return int32(uint32(row[i]) | uint32(row[i+1])<<8 | uint32(row[i+2])<<16 | uint32(row[i+3])<<24)
}

// Hit points gained per level once a class is past its hit dice.
const (
	fixedHPFighter = 3 // fighter, paladin
	fixedHPOther   = 2 // ranger, cleric, thief
	fixedHPMage    = 1
)

// rollHitPoints2 = O0A7:49E0 (c, mask): hit points rolled for the classes of
// the mask; each class rolls its dice twice and keeps the better. A class
// past its hit dice sets (not adds) a fixed amount (original quirk).
func (e *Engine) rollHitPoints2(c *Character, mask int) int {
	total := byte(0)
	for k := 0; k <= clsMonk; k++ {
		lvl := int8(c.Levels[k])
		if lvl <= 0 || classBit(k)&byte(mask) == 0 {
			continue
		}
		if byte(lvl) < classMaxHitDice[k] {
			count := int(classHitDiceFirst[k])
			if lvl > 1 {
				count = 1
			}
			a := byte(e.Dice(count, int(classHitDiceSides[k])))
			b := byte(e.Dice(count, int(classHitDiceSides[k])))
			total += max(a, b)
			continue
		}
		switch k {
		case clsFighter, clsPaladin:
			total = fixedHPFighter
		case clsRanger, clsCleric, clsThief:
			total = fixedHPOther
		case clsMage:
			total = fixedHPMage
		}
	}
	return int(total)
}

// conHitPointBonusOf = O0A7:45C1: the constitution hit point bonus summed
// over the classes still rolling hit dice; fighters, paladins and rangers
// get more for high constitution, a first-level ranger doubles it.
func (e *Engine) conHitPointBonusOf(c *Character) int {
	bonus := int8(0)
	con := c.Stats[statCon].Full
	for k := 0; k <= clsMonk; k++ {
		lvl := int8(c.Levels[k])
		if lvl <= 0 || byte(lvl) >= classMaxHitDice[k] {
			continue
		}
		bonus += conHitPointBonus[con]
		if c.Class == clsFighter || c.Class == clsPaladin || c.Class == clsRanger {
			switch {
			case con == 17:
				bonus++
			case con == 18:
				bonus += 2
			case con == 19 || con == 20:
				bonus += 3
			case con >= 21 && con <= 23:
				bonus += 4
			case con == 24 || con == 25:
				bonus += 5
			}
		}
		if k == clsRanger && c.Levels[clsRanger] == 1 {
			bonus *= 2
		}
	}
	return int(bonus)
}

// goldWorth = O0AF:33AE getPlayerGold: the character's coins in gold
// pieces (rounded by +100 copper).
func goldWorth(c *Character) int32 {
	var copper int32
	for k := range coinValues {
		copper += coinValues[k] * int32(c.Coins[k])
	}
	return (copper + 100) / coinValues[coinGold]
}

// payGold = O0A7:4AE0 (gold, c): take gold pieces' worth from the coins,
// smallest coins first; change is given back from platinum down.
func payGold(c *Character, gold int32) {
	copper := gold * coinValues[coinGold]
	k := 0
	for copper > 0 {
		n := uint16(copper/coinValues[k] + 1)
		n = min(n, c.Coins[k])
		copper -= int32(n) * coinValues[k]
		c.Coins[k] -= n
		k++
	}
	if copper < 0 {
		copper = -copper
		for k = coinPlatinum; copper > 0; k-- {
			n := uint16(copper / coinValues[k])
			copper -= int32(n) * coinValues[k]
			c.Coins[k] += n
		}
	}
}

// Coin indexes.
const (
	_ = iota // coinCopper (money.go holds the copper worth table of the same name)
	coinSilver
	coinElectrum
	coinGold
	coinPlatinum
	coinGems
	coinJewelry
)

const trainingCost = 1000 // gold pieces

// Spells a magic-user learns while being trained silently (new characters
// start above level 1), by the magic-user level reached.
var silentMageSpells = map[byte][]int{
	2: {spellMagicMissile},
	3: {spellStinkingCloud, spellCharmPerson},
	4: {spellKnock},
	5: {spellFireball},
}

// trainSilently = the creation's training loop: train without questions,
// costs or messages until no class can advance any more.
func (e *Engine) trainSilently(c *Character) {
	undo := e.selectTemp(c)
	defer undo()
	savedMask := e.area[trainingMaskByte]
	e.area[trainingMaskByte] = 255
	e.cannotTrain = false
	e.silentTraining = true
	for !e.cannotTrain {
		e.TrainCharacter()
	}
	e.silentTraining = false
	e.area[trainingMaskByte] = savedMask
}

// trainingMaskByte: the area byte holding the classes the local trainer teaches.
const trainingMaskByte = 1360

// TrainCharacter = O0A7:4D07 train_player: raise every class of the selected
// character whose experience allows it and the local trainer teaches.
func (e *Engine) TrainCharacter() {
	c := e.selected()
	if c.Status != 0 && !e.freeTraining {
		e.statusMessage("we only train conscious people", colourYellow, colourBlack)
		return
	}
	if goldWorth(c) < trainingCost && !e.freeTraining && !e.silentTraining && !e.gameWon {
		e.statusMessage("Training costs 1000 gp.", colourYellow, colourBlack)
		return
	}
	var ready, have byte // classes with enough experience, classes the character has
	var cap int32        // the experience to keep: just short of the second level up
	trainer := e.area[trainingMaskByte]
	lastLevel := 0
	for k := 0; k <= clsMonk; k++ {
		lvl := int8(c.Levels[k])
		if lvl <= 0 {
			continue
		}
		have += classBit(k)
		lastLevel = int(lvl)
		if e.atRaceLimit(c, k, int(lvl)) {
			continue
		}
		need := expThreshold(k, int(lvl))
		if !(need <= int32(c.Exp) || e.freeTraining) || need <= 0 {
			continue
		}
		if e.freeTraining && int32(c.Exp) < need {
			c.Exp = uint32(need)
		}
		ready += classBit(k)
		if next := expThreshold(k, int(lvl)+1); next > 0 && next <= int32(c.Exp) && next > cap {
			cap = next - 1
		}
	}
	if !e.silentTraining {
		// Only the class with the highest threshold advances at a time.
		var best int32
		bestClass := 0
		for k := 0; k <= clsMonk; k++ {
			if classBit(k)&ready != 0 {
				if t := expThreshold(k, lastLevel); t > best {
					best, bestClass = t, k
				}
			}
		}
		if best > 0 {
			ready = classBit(bestClass)
			if next := expThreshold(bestClass, lastLevel+1); next > 0 && next <= int32(c.Exp) && next > cap {
				cap = next - 1
			}
		}
	}
	if cap > 0 && !e.silentTraining {
		c.Exp = uint32(cap)
	}
	if !e.freeTraining {
		if have&trainer == 0 && !e.silentTraining {
			e.statusMessage("We don't train that class here", colourYellow, colourBlack)
			return
		}
		if ready&trainer == 0 {
			e.cannotTrain = true
			if !e.silentTraining {
				e.statusMessage("Not Enough Experience", colourYellow, colourBlack)
			}
			return
		}
	}
	training := ready
	if !e.freeTraining {
		training &= trainer
	}
	if !e.silentTraining {
		e.ClearCells(1, 1, 38, 22)
		row := 4
		e.displayName(c, 4, row, false)
		e.displayStringBg(len(c.Name())+4, row, colourGreen, colourBlack, " will become:")
		for k := 0; k <= clsMonk; k++ {
			if int8(c.Levels[k]) > 0 && classBit(k)&training != 0 {
				row++
				lead := "and a level "
				if row == 5 {
					lead = "    a level "
				}
				e.displayStringBg(6, row, colourGreen, colourBlack, lead+itoa(int(c.Levels[k])+1)+" "+classNames[k])
			}
		}
		if e.YesNo(colourLightMagenta, colourGreen, colourWhite, "Do you wish to train? ") != 'Y' {
			return
		}
		e.printMessage("Congratulations...")
		if !e.freeTraining && !e.gameWon {
			payGold(c, trainingCost)
		}
	}
	classes := 0
	oldMage := c.Levels[clsMage]
	c.ClassMask = 0
	for k := 0; k <= clsMonk; k++ {
		if int8(c.Levels[k]) <= 0 {
			continue
		}
		classes++
		if classBit(k)&training != 0 {
			c.Levels[k]++
			if c.Drained != 0 {
				c.DrainedHP -= c.DrainedHP / c.Drained
				c.Drained--
			}
		}
	}
	e.recalcClassBonuses(c)
	if !e.silentTraining {
		if int8(oldMage) < int8(c.Levels[clsMage]) || int8(c.Levels[clsRanger]) > rangerSpellLevel {
			e.learnNewSpell(c)
		}
	} else {
		for _, s := range silentMageSpells[c.Levels[clsMage]] {
			c.SpellBook[s-1] = 1
		}
	}
	if int8(c.MulticlassLevel) >= int8(c.MaxLevel) {
		return
	}
	rolled := e.rollHitPoints2(c, int(training)) & 0xFF
	gain := rolled / classes
	if gain == 0 {
		gain = 1
	}
	c.HPRoll += byte(gain)
	gain = (rolled + e.conHitPointBonusOf(c)) / classes
	if gain < 1 {
		gain = 1
	}
	wounds := c.MaxHP - c.HP
	c.MaxHP += byte(gain)
	c.HP = c.MaxHP - wounds
}

// rangerSpellLevel: rangers above this level learn magic-user spells.
const rangerSpellLevel = 8

// atRaceLimit = the race checks in O0A7:4D07: demi-humans stop at a
// maximum level per class, some only with a high strength or intelligence.
func (e *Engine) atRaceLimit(c *Character, class, level int) bool {
	str, intl := c.Stats[statStr].Full, c.Stats[statInt].Full
	limit := func(max, ifStr17 int, ability byte, highOnly bool) bool {
		// at max always; at max-1 with ability 17; at max-2 with ability under 17
		return level == max || level == max-1 && ability == 17 || level == max-2 && ability < 17 && highOnly
	}
	switch int(c.Race) {
	case raceDwarf:
		return class == clsFighter && limit(9, 0, str, true)
	case raceElf:
		return class == clsFighter && limit(7, 0, str, true) ||
			class == clsMage && (level == 11 || level == 9 && intl < 17 || level == 10 && intl == 17)
	case raceGnome:
		return class == clsFighter && (level == 6 || level == 5 && str < 18)
	case raceHalfElf:
		if class == clsCleric && level == 5 {
			return true
		}
		return (class == clsFighter || class == clsRanger) && limit(8, 0, str, true) ||
			class == clsMage && limit(8, 0, intl, true)
	case raceHalfling:
		return class == clsFighter && limit(6, 0, str, true)
	}
	return false
}
