package engine

// Full-screen frames drawn from 8x8 symbols (unit R133 of the reconstruction).
// The frame tables (tables.go) hold one symbol per cell, offset by
// frameSymbolBase.

const (
	frameSymbolBase = 286
	screenLastCol   = 39
	screenLastRow   = 23
	creditsRuleRow1 = 3
	creditsRuleRow2 = 8
)

// drawScreenFrame = R133:0000 draw8x8_01: clear the inside and draw the
// border around the whole screen.
func (e *Engine) drawScreenFrame() {
	e.ClearCells(1, 1, screenLastCol-1, screenLastRow-1)
	for x := 0; x <= screenLastCol; x++ {
		e.Put8x8Symbol(x, 0, int(frameTop[x])+frameSymbolBase)
	}
	for y := 0; y <= screenLastRow; y++ {
		e.Put8x8Symbol(0, y, int(frameLeft[y])+frameSymbolBase)
		e.Put8x8Symbol(screenLastCol, y, int(frameRight[y])+frameSymbolBase)
	}
	for x := 0; x <= screenLastCol; x++ {
		e.Put8x8Symbol(x, screenLastRow, int(frameBottom[x])+frameSymbolBase)
	}
}

// drawCreditsFrame = R133:00D3 draw8x8_02: the screen frame with two rules
// across it, under the novel's credit and under the scenario credit.
func (e *Engine) drawCreditsFrame() {
	e.drawScreenFrame()
	for x := 0; x <= screenLastCol; x++ {
		e.Put8x8Symbol(x, creditsRuleRow1, int(frameMiddleRow[x])+frameSymbolBase)
		e.Put8x8Symbol(x, creditsRuleRow2, int(frameMiddleRow[x])+frameSymbolBase)
	}
}
