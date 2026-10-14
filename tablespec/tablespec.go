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
	Len  int // elements
	Kind Kind
	Doc  string
}

// Tables are generated in this order.
var Tables = []Table{
	// Screen frames drawn from 8x8 symbols (R133 draw8x8_01..): symbol = entry + 286.
	{"frameTop", 2896, 40, Bytes, "Frame symbols along the top row (y 0), one per column."},
	{"frameMiddleRow", 2936, 40, Bytes, "Frame symbols of the separator rows of the credits and the 3D view frame."},
	{"frameBottom", 2976, 40, Bytes, "Frame symbols along row 23."},
	{"frameLeft", 3042, 24, Bytes, "Frame symbols down the left column (x 0)."},
	{"frameRight", 3083, 24, Bytes, "Frame symbols down the right column (x 39)."},
}
