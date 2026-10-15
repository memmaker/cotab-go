package engine

// Appraise = ovr22:19B2 (var any): the shop and temple "Appraise" command.
// Each of the party's gems and pieces of jewelry is valued and sold or kept
// as an item. any is false when there is nothing to show.
func (e *Engine) Appraise() (any bool) {
	any = true
	c := e.selected()
	p := &e.purse
	if p.gems == 0 && p.jewelry == 0 {
		e.BottomMessage("No Gems or Jewelry")
		return false
	}
	for done := false; !done; {
		if p.gems == 0 && p.jewelry == 0 {
			done = true
			continue
		}
		count := func(n int, one, many string) string {
			switch n {
			case 0:
				return ""
			case 1:
				return cut(itoa(n)+one, 0x28)
			}
			return cut(itoa(n)+many, 0x28)
		}
		gems := count(p.gems, " Gem", " Gems")
		jewels := count(p.jewelry, " piece of Jewelry", " pieces of Jewelry")
		e.ClearBox(1, 1, 0x26, 0x16)
		e.displayName(c, 1, 1, false) // 00BA:1816
		e.DisplayString(1, 7, 0xF, "You have a fine collection of:")
		e.DisplayString(1, 9, 0xF, gems)
		e.DisplayString(1, 0xA, 0xF, jewels)
		menu := ""
		if p.gems != 0 {
			menu = "  Gems"
		}
		if p.jewelry != 0 {
			menu = cut(menu+"  Jewelry", 0x28)
		}
		menu = cut(menu+" Exit", 0x28)
		key, _ := e.DisplayInput("Appraise : ", menu, 0xD, 0xA, 0xF, true, false)
		switch key {
		case 'G':
			if p.gems == 0 {
				break
			}
			p.gems--
			var v uint16
			switch d := e.Dice(1, 100); {
			case d <= 25:
				v = 10
			case d <= 50:
				v = 50
			case d <= 70:
				v = 100
			case d <= 90:
				v = 500
			case d <= 99:
				v = 1000
			default:
				v = 5000
			}
			e.DisplayString(1, 0xC, 0xF, cut("The Gem is Valued at "+itoa(int(v))+" gp.", 0x28))
			e.sellOrKeep(c, v, 0x65)
		case 'J':
			if p.jewelry == 0 {
				break
			}
			p.jewelry--
			var v uint16
			switch d := e.Dice(1, 100); {
			case d <= 10:
				v = uint16(e.Random(900) + 100)
			case d <= 20:
				v = uint16(e.Random(1000) + 200)
			case d <= 40:
				v = uint16(e.Random(1500) + 300)
			case d <= 50:
				v = uint16(e.Random(2500) + 500)
			case d <= 70:
				v = uint16(e.Random(5000) + 1000)
			case d <= 90:
				v = uint16(e.Random(6000) + 2000)
			default:
				v = uint16(e.Random(10000) + 2000)
			}
			e.DisplayString(1, 0xC, 0xF, cut("The Jewel is Valued at "+itoa(int(v))+" gp.", 0x28))
			e.sellOrKeep(c, v, 0xD6)
		case 'E', 0:
			done = true
		}
		e.ApplyItems(c)
	}
	return any
}

// sellOrKeep is the shared tail of the gem and jewel branches of 19B2:
// "Sell Keep" when the character can carry one more item, else only Sell.
func (e *Engine) sellOrKeep(c *Character, v uint16, word byte) {
	sellOnly := c.ItemCount >= 0x10
	menu := "Sell Keep"
	if sellOnly {
		menu = "Sell"
	}
	key, _ := e.DisplayInput("You can : ", menu, 0xD, 0xA, 0xF, true, false)
	if key == 'K' && !sellOnly {
		it := &Item{}
		it.Words[2] = word
		it.Type = 0x46
		it.Price = v
		it.Weight = 1
		it.Hidden = 0
		// The original appends to the item list but, when the list is empty,
		// only to its local copy: the gem is lost. Kept as written.
		if len(c.Items) > 0 {
			c.Items = append(c.Items, it)
		}
		return
	}
	e.purse.gold += int(v)
}
