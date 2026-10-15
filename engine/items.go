package engine

// Items: names, lists and the START helpers around them.

// itemWord: the n-th word item names are made of.
func (e *Engine) itemWord(n byte) string { return itemWords[n] }

func (it *Item) Name() string {
	n := int(it.PName[0])
	if n > 0x28 {
		n = 0x28
	}
	return string(it.PName[1 : n+1])
}

func (it *Item) setName(s string) {
	s = cut(s, 0x28)
	it.PName[0] = byte(len(s))
	copy(it.PName[1:41], s)
}

// itemStats (extra): Angband-style base stats after the name, "(1d8) "
// for weapon damage, "<6> " for the armour class it gives (the font has
// no square brackets); magic pluses stay in the name.
func (e *Engine) itemStats(typ byte) string {
	s := ""
	if n, d := e.baseItem(typ, 9), e.baseItem(typ, 0xA); n > 0 && d > 0 {
		s = "(" + itoa(int(n)) + "d" + itoa(int(d))
		if b := int8(e.baseItem(typ, 0xB)); b > 0 {
			s += "+" + itoa(int(b))
		} else if b < 0 {
			s += itoa(int(b))
		}
		s += ") "
	}
	if v := int(e.baseItem(typ, 6)); v > 0x80 {
		if v &= 0x7F; v >= 50 {
			v -= 50 // body armour counts up from AC 10 = 50
		}
		if v > 0 {
			s += "<" + itoa(v) + "> "
		}
	}
	return s
}

// RemoveItem = 00BA:1509 (char, item): unlinks the item and frees it.
func (e *Engine) RemoveItem(c *Character, it *Item) {
	if it == nil {
		e.MessageRow24b(0xE, "Nil Item pointer...")
		return
	}
	for i, x := range c.Items {
		if x == it {
			c.Items = append(c.Items[:i:i], c.Items[i+1:]...)
			return
		}
	}
	e.MessageRow24b(0xE, "Tried to Lose item & couldn't find it!")
}

// AddItemCopy = 00BA:15F7 (char, item): a copy of the item at the end of
// the character's list.
func (e *Engine) AddItemCopy(c *Character, it *Item) *Item {
	n := *it
	c.Items = append(c.Items, &n)
	return &n
}

// BottomMessage = 00BA:1693: the string on row 24 for the message delay.
func (e *Engine) BottomMessage(s string) {
	s = cut(s, 0x28)
	e.ClearCells(0, 24, 0x27, 24)
	e.DisplayString(0, 24, 0xA, s)
	e.MsgDelay()
	e.ClearCells(0, 24, 0x27, 24)
}
