// Package tablespec lists the data tables cmd/tablegen copies out of
// START.EXE: name, place in the data segment (decimal offsets, as in the
// reconstruction's GAMEDATA unit), length and kind. Only the generator uses it.
package tablespec

// Kind is the element type of a table.
type Kind int

const (
	Bytes Kind = iota
	Signed
	Words
	SignedWords
	// Strings of a fixed slot size: String(n) holds a length byte and n-1 characters.
	stringBase
)

// String is a table of Pascal strings in slots of size bytes.
func String(size int) Kind { return stringBase + Kind(size) }

// Size is the bytes per element.
func (k Kind) Size() int {
	switch k {
	case Bytes, Signed:
		return 1
	case Words, SignedWords:
		return 2
	}
	return int(k - stringBase)
}

// Table is one generated Go value.
type Table struct {
	Name string
	At   int // offset in the data segment
	Len  int // elements (rows when Cols is set)
	Kind Kind
	Doc  string
	Cols int // set for a two-dimensional table: Len rows of Cols elements
}

// Tables are generated in this order.
var Tables = []Table{
	// Screen frames drawn from 8x8 symbols (R133 draw8x8_01..): symbol = entry + 286.
	{"frameTop", 2896, 40, Bytes, "Frame symbols along the top row (y 0), one per column.", 0},
	{"frameMiddleRow", 2936, 40, Bytes, "Frame symbols of the separator rows of the credits and the 3D view frame.", 0},
	{"frameBottom", 2976, 40, Bytes, "Frame symbols along row 23.", 0},
	{"frameLeft", 3042, 24, Bytes, "Frame symbols down the left column (x 0).", 0},
	{"frameRight", 3083, 24, Bytes, "Frame symbols down the right column (x 39).", 0},

	// Names.
	{"raceNames", 3742, 8, String(10), "Race names: monster, dwarf, elf, gnome, half-elf, halfling, half-orc, human.", 0},
	{"genderNames", 3975, 2, String(7), "Gender names.", 0},
	{"classNames", 3256, 17, String(27), "Class names; 8.. are the multi-class combinations.", 0},
	{"alignmentNames", 3822, 9, String(17), "Alignment names.", 0},

	// Character creation (O0A7 createPlayer).
	{"defaultIconColours", 16067, 6, Bytes, "Combat icon colours of a new character, one per icon part.", 0},
	{"raceClasses", 16378, 8, Bytes, "Per race: the number of classes it may take, then those classes.", 14},
	{"classAlignments", 16858, 17, Bytes, "Per class: the number of alignments it allows, then those alignments.", 10},
	{"thac0Table", 15930, 8, Bytes, "Per class and level (0..12): the THAC0 base value.", 13},
	{"classItemFlags", 16034, 8, Bytes, "Per class: the item-use bits it adds to the class mask.", 0},
	{"raceAges", 16462, 8, Bytes, "Per race, for the classes 0..6: starting age base (word), dice count, dice sides.", 28},
	{"ageSteps", 16676, 8, Words, "Per race: the ages at which abilities change (four steps, one spare word).", 5},
	{"raceStatLimits", 16264, 8, Bytes, "Per race: strength min/max by gender, exceptional strength max by gender, then min/max of int wis dex con cha.", 16},
	{"classStatMinimum", 16756, 17, Bytes, "Per class: the minimum of each ability (str int wis dex con cha).", 6},
	{"classHitDice", 17843, 8, Bytes, "Per class: dice count and sides of the starting hit points.", 2},

	// Spells.
	{"spellTableData", 14300, 101, Bytes, "The spell table: one 16-byte record per spell number (spellEntry).", 16},
}
