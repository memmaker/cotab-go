package engine

// ovr20: the View screen (character sheet, items, money, spell lists).

// ClearScreen = 0709:0CAB.
func (e *Engine) ClearScreen() {
	if e.Trace != nil {
		e.Trace("0709:0CAB", "ClearScreen")
	}
	e.fillCells(0, 0, 0x27, 0x18)
}

// viewChar = DS:4716, the character on the sheet.
func (e *Engine) viewChar() *Character { return e.view.char }

type viewState struct {
	char  *Character // DS:4716
	start *Character // DS:471A
}

// inventory runs ItemsMenu for ViewCharacter (extra:
// leave reports that the View page should close too: F1-F6 switched to
// another member, or Esc in an inventory F1-F6 opened, unless P came back
// to the View page).
func (e *Engine) inventory(direct bool) (quit, leave bool) {
	e.ext.toSheet, e.ext.switchTo = false, false
	quit = e.ItemsMenu()
	leave = !quit && (e.ext.switchTo || direct && !e.ext.toSheet)
	e.ext.toSheet, e.ext.switchTo = false, false
	return quit, leave
}

// canLoseItem = ovr20:0D70 (item): readied items must be unreadied first;
// a scroll someone is about to scribe from asks before it goes.
func (e *Engine) canLoseItem(it *Item) bool {
	ok := false
	switch {
	case it.Readied != 0:
		e.BottomMessage("Must be unreadied")
	case !e.isScroll(it):
		ok = true
	case it.Spells[0] > 0x7F || it.Spells[1] > 0x7F || it.Spells[2] > 0x7F:
		c := e.selected()
		e.displayName(c, 1, 0x15, false)
		e.msgCol, e.msgRow = len(c.Name())+2, 0x15
		e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, false, " was going to scribe from that scroll")
		ok = e.YesNo(0xD, 0xA, 0xF, "is it Okay to lose it? ") == 'Y'
	default:
		ok = true
	}
	e.ClearCells(1, 0x15, 0x26, 0x16)
	return ok
}

// SheetPortrait = ovr29:04F9: the character's head and body in a box at
// the top right of the sheet.
func (e *Engine) SheetPortrait() {
	c := e.selected()
	if c.PortHead == 0 || c.PortBody == 0 {
		return
	}
	e.DrawBoxTitled(0x1C, 1, 0x26, 0xB, 0, 0, "")
	e.LoadPortrait(int(e.X.Bytes(0x286F+int(c.PortHead), 1)[0]), int(e.X.Bytes(0x287D+int(c.PortBody), 1)[0]))
	e.DrawPortrait(0x1C, 1, true)
}

// ItemsMenu = O0AF:1583 use_item (var done): the character's items with
// Ready, Use, Trade, Drop, Halve, Join, Sell and Id. Returns done (an item
// was used in combat).
func (e *Engine) ItemsMenu() bool {
	c := e.selected()
	key := byte(' ')
	index := 0
	redraw, full := true, true
	done := false
	for key != 0 && key != 'E' && !done && c.ItemCount > 0 {
		count := c.ItemCount
		if len(c.Items) > 0 {
			menu := "Ready"
			if c.Okay != 0 && wordAt(e.area, 2*areaWordNoMagic) == 0 &&
				(e.gameState == stateCamp || e.gameState == stateWilderness || e.gameState == stateDungeon ||
					e.gameState == stateCombat && c.Combat != nil && c.Combat.CanUse != 0) {
				menu += " Use"
			}
			hired := c.Control >= npcControl && c.Okay != 0 && c.Status != statusAnimated
			if !hired && e.gameState != stateCombat {
				menu += " Trade"
			}
			menu += " Drop"
			if c.ItemCount < maxItems {
				menu += " Halve"
			}
			menu += " Join"
			if !hired && e.gameState == stateShop {
				menu += " Sell"
			}
			if e.gameState == stateShop {
				menu += " Id"
			}
			list := make([]*ListNode, len(c.Items))
			for i, x := range c.Items {
				e.ItemName(c, x, 0, 0, true, false)
				list[i] = &ListNode{Text: x.Name()}
			}
			if full || e.itemsRedraw {
				e.drawListFrame()
				e.displayName(c, 1, 1, true)
				e.displayStringBg(int(c.PName[0])+4, 1, colourGreen, colourBlack, "Items")
				e.displayStringBg(1, 3, colourWhite, colourBlack, "Ready Item")
				redraw, full, e.itemsRedraw = true, false, false
			}
			var sel *ListNode
			key = e.ListMenu("", menu, colourLightMagenta, colourGreen, colourWhite, 1, 5, 38, 22, list, true, &redraw, &index, &sel)
			var it *Item
			for i, n := range list {
				if n == sel {
					it = c.Items[i]
				}
			}
			if it != nil {
				switch key {
				case 'V':
					e.itemDebug(it)
					redraw, full = true, true
				case 'R':
					e.readyItem(it)
				case 'U':
					if it.Readied == 0 {
						e.printMessage("Must be Readied")
						key = ' '
					} else if e.isScroll(it) || it.Spells[1] > 0 && it.Spells[2] < 0x80 {
						done = e.useItem(it)
						if e.gameState != stateCombat {
							done = false
						}
						if !done {
							full = true
						}
					}
				case 'T':
					if e.canLoseItem(it) {
						e.tradeItem(it)
					} else {
						key = ' '
					}
					full = true
				case 'D':
					if e.canLoseItem(it) {
						e.ItemName(c, it, 0, 0, false, false)
						e.msgCol, e.msgRow = 1, 21
						e.DisplayTextBlock(1, 21, 38, 22, colourYellow, true, "Your "+it.Name()+"will be gone forever")
						if e.YesNo(colourLightMagenta, colourGreen, colourWhite, "Drop It? ") == 'Y' {
							e.loseItem(e.selected(), it)
							redraw = true
						}
						e.ClearCells(1, 21, 38, 22)
					} else {
						key = ' '
					}
				case 'H':
					e.halveItem(it)
				case 'J':
					e.joinItems(it)
				case 'S':
					if e.canLoseItem(it) {
						e.sellItem(it)
					} else {
						key = ' '
					}
				case 'I':
					if e.identifyItem(it) {
						redraw = true
					}
				}
			}
			e.ApplyItems(c)
		}
		if c.ItemCount != count {
			redraw = true
		}
	}
	return done
}

// itemDebug = O0AF:10A6: the item record's fields (a debugging page).
func (e *Engine) itemDebug(it *Item) {
	e.drawScreenFrame()
	labels := []string{"itemptr:      ", "namenum(1):   ", "namenum(2):   ", "namenum(3):   ", "plus:         ",
		"plussave:     ", "ready:        ", "identified:   ", "cursed:       ", "value:        ", "special(1):   ",
		"special(2):   ", "special(3):   ", "dice large:    ", "sides large:   "}
	for i, l := range labels {
		e.displayStringBg(1, i+1, colourGreen, colourBlack, l)
	}
	values := []int{int(it.Type), int(it.Words[0]), int(it.Words[1]), int(it.Words[2]), int(it.Plus), int(it.Plus2),
		int(it.Readied), int(it.Hidden), int(it.Cursed), int(it.Price), int(it.Spells[0]), int(it.Spells[1]), int(it.Spells[2]),
		int(e.baseItem(it.Type, baseDiceLarge)), int(e.baseItem(it.Type, baseSidesLarge))}
	for i, v := range values {
		e.displayStringBg(20, i+1, colourGreen, colourBlack, itoa(v))
	}
	e.displayAndWait(colourGreen, colourBlack, "press a key")
}

// Base item record fields itemDebug shows.
const (
	baseDiceLarge  = 2
	baseSidesLarge = 3
)

// readyItem = ovr20:14D0 (item): ready or unready, with the reasons it
// cannot be readied and the item's own effect (+0x3E > 0x7F).
func (e *Engine) readyItem(it *Item) {
	effect := it.Spells[2] > 0x7F
	c := e.selected()
	if it.Readied != 0 {
		if it.Cursed != 0 {
			e.BottomMessage("It's Cursed")
			return
		}
		it.Readied = 0
		if effect {
			e.itemAffect(it, c, 1)
		}
		return
	}
	why := 0
	typ := it.Type
	if int(e.baseItem(typ, 1))+int(c.Hands) > 2 {
		why = 3
	}
	slot := int(e.baseItem(typ, 0))
	switch {
	case slot <= 8:
		if c.Slots[slot] != nil {
			why = 2
		}
	case slot == 9:
		if c.Slots[slotRing2] != nil {
			why = 2
		}
	}
	if typ == 0x49 && c.Slots[slotArrows] != nil {
		why, slot = 2, slotArrows
	}
	if typ == 0x1C && c.Slots[slotBolts] != nil {
		why, slot = 2, slotBolts
	}
	if e.baseItem(typ, 0xD)&c.ClassMask == 0 {
		why = 1
	}
	switch why {
	case 0:
		it.Readied = 1
		if effect {
			e.itemAffect(it, c, 0)
		}
	case 1:
		e.BottomMessage("Wrong Class")
	case 2:
		// slot 9 (ring 2 taken) names Slots[9], which may be empty: the
		// original then reads through a nil pointer.
		name := ""
		if o := c.Slots[slot]; o != nil {
			e.ItemName(c, o, 0, 0, false, false)
			name = o.Name()
		}
		e.BottomMessage("already using " + name)
	case 3:
		if !(e.gameState == 5 && c.Auto != 0) {
			e.BottomMessage("Your hands are full!")
		}
	}
}

// itemAffect calls ovr25:0000 with the item standing in for the affect
// record, as the original does.
func (e *Engine) itemAffect(it *Item, c *Character, flag byte) {
	a := &Affect{Item: it}
	// the affect fields are the item's first bytes, as in the original
	n := it.PName
	a.Type, a.Time, a.Data, a.Call = n[0], uint16(n[1])|uint16(n[2])<<8, n[3], n[4]
	e.callAffect(it.Spells[2], c, a, flag)
}

// tradeItem = ovr20:1734 (item).
func (e *Engine) tradeItem(it *Item) {
	to := e.view.start
	e.DrawMainScreen()
	to = e.SelectCharacter("Trade with Whom?", true, to)
	if to == nil {
		return
	}
	e.view.start = to
	if e.overloaded(it, to) {
		e.BottomMessage("Overloaded")
		return
	}
	e.AddItemCopy(to, it)
	e.RemoveItem(e.selected(), it)
	e.ApplyItems(to)
}

// halveItem = ovr20:17ED (item): half the stack becomes a new unreadied
// item right after it.
func (e *Engine) halveItem(it *Item) {
	half := it.Count >> 1
	if half == 0 {
		e.BottomMessage("Can't halve that")
		return
	}
	n := *it
	n.Count = half
	n.Readied = 0
	it.Count -= half
	c := e.selected()
	for i, x := range c.Items {
		if x == it {
			c.Items = append(c.Items[:i+1], append([]*Item{&n}, c.Items[i+1:]...)...)
			return
		}
	}
}

// joinItems = ovr20:189D (item): every matching stack joins it, up to 255
// a stack. (The original compares +0x3C..+0x3E of each item with itself.)
func (e *Engine) joinItems(it *Item) {
	total := int(it.Count)
	c := e.selected()
	for _, p := range append([]*Item(nil), c.Items...) {
		if p == it || p.Count == 0 {
			continue
		}
		same := p.Words == it.Words && p.Type == it.Type && p.Plus == it.Plus && p.Plus2 == it.Plus2 && p.Cursed == it.Cursed
		if !same || p.Weight != it.Weight || p.Spells[0] >= 2 {
			continue
		}
		if int(p.Count)+total <= 0xFF {
			total += int(p.Count)
			e.RemoveItem(c, p)
		} else {
			it.Count = 0xFF
			p.Count -= byte(0xFF - total)
			total = int(p.Count)
			it = p
		}
	}
	it.Count = byte(total)
}

// useItem = ovr20:1A8A (item): cast the scroll's spell or the item's own
// (+0x3D, charges +0x3C). Returns the spell's "turn used" result.
func (e *Engine) useItem(it *Item) bool {
	idx := -1
	e.usingItem = false
	var spell byte
	quit := false
	if e.isScroll(it) {
		e.usedScroll = it
		var flag bool
		spell = e.SpellList(2, 1, &idx, &flag)
	} else if it.Spells[1] > 0 && it.Spells[2] < 0x80 {
		e.usingItem = true
		spell = it.Spells[1]
	}
	if spell == 0 {
		return false
	}
	c := e.selected()
	if e.gameState == 5 && c.Auto == 0 {
		e.combatUseHook() // 00BA:2B88
	}
	if e.usingItem {
		if spell > 0x38 {
			spell -= 0x17
		}
		e.CharMessage(c, "uses an item", 0xA, false)
		if e.gameState == 5 {
			e.DisplayString(0, 0x17, 0xA, "Item:")
			e.ItemName(c, it, 5, 0x17, false, true)
		} else {
			e.ItemName(c, it, 1, 0x16, false, true)
		}
		e.MsgDelay()
		e.ClearMessageArea()
	}
	e.usingItem = true
	e.CastSpell(spell, c.Auto, 0, &quit)
	e.usingItem = false
	if e.gameState == 5 && e.spellRec(spell, 0xB) != 0 {
		quit = e.endTurn(c)
	}
	if quit {
		if e.isScroll(it) {
			e.scrollUsed(c, it, spell)
		} else if it.Spells[0] > 0 {
			if it.Count > 1 {
				it.Count--
			} else {
				it.Spells[0]--
				if it.Spells[0] == 0 {
					e.RemoveItem(c, it)
				}
			}
		}
	}
	return quit
}

// spellRec = DS:31FA + 16*spell.
func (e *Engine) spellRec(spell byte, k int) byte {
	return e.X.Bytes(0x31FA+int(spell)*16+k, 1)[0]
}

// sellItem = ovr20:1CEE (item): half the value (a twentieth per piece
// of a stack), paid into the party's gold.
func (e *Engine) sellItem(it *Item) {
	v := it.Price >> 1
	if it.Count > 1 {
		// (the arrows/quarrels branch of the original can never be taken)
		v = uint16(uint32(uint16(uint32(it.Count)*uint32(v))) / 0x14)
	}
	e.ItemName(e.viewChar(), it, 0, 0, false, false)
	e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true,
		"I'll give you "+itoa(int(v))+" gold pieces for your "+it.Name())
	if e.YesNo(0xD, 0xA, 0xF, "Is It a Deal? ") == 'Y' {
		e.BottomMessage("Sold!")
		e.RemoveItem(e.selected(), it)
		e.purse.gold += int(v)
	}
	e.ClearCells(1, 0x15, 0x26, 0x16)
}

// identifyItem = ovr20:1F52 (item, redraw): 200 gp shows the hidden words.
func (e *Engine) identifyItem(it *Item) (redraw bool) {
	e.ItemName(e.viewChar(), it, 0, 0, false, false)
	e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "For 200 gold pieces I'll identify your "+it.Name())
	if e.YesNo(0xD, 0xA, 0xF, "Is It a Deal? ") == 'Y' && e.pay(200) {
		if it.Hidden == 0 {
			e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "I can't tell anything new about your "+it.Name())
		} else {
			it.Hidden = 0
			e.ItemName(e.viewChar(), it, 0, 0, false, false)
			e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "It looks like some sort of "+it.Name())
			redraw = true
		}
		e.MsgDelay()
	}
	e.ClearCells(1, 0x15, 0x26, 0x16)
	return
}

// overloaded = ovr20:275B (item, char): no room for one more item.
func (e *Engine) overloaded(it *Item, c *Character) bool {
	e.ApplyItems(c)
	over := c.ItemCount > 0xF
	w := it.Weight
	if it.Count > 0 {
		w *= uint16(it.Count)
	}
	if int32(c.Weight+w) > int32(strCarry(c))+1500 {
		over = true
	}
	return over
}

// SpellList = ovr20:299C (kind, mode, index, any): the list of spells of
// one kind (0 memory, 1 book, 2 scroll, 3 scrolls, 4 choose, 5 to be
// memorised, 6 to be scribed); returns the chosen spell or 0.
func (e *Engine) SpellList(kind, mode int, index *int, any *bool) byte {
	titles := []string{"in Memory", "in Spell Book", "on Scroll", "on Scrolls", "to choose from", "to be memorized", "to be scribed"}
	title := ""
	if kind >= 0 && kind < len(titles) {
		title = titles[kind]
	}
	*any = e.buildSpellList(kind)
	if !*any {
		return 0
	}
	if *index < 0 {
		e.DrawBoxTitled(1, 1, 0x26, 1, 0, 0, "")
		c := e.selected()
		e.displayName(c, 1, 1, true)
		e.DisplayString(len(c.Name())+4, 1, 0xA, "Spells "+title)
	}
	return e.spellMenu(mode, index)
}
