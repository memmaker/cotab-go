package engine

// The view-window animation that runs while a menu waits (O110:000A and
// the frame table DS 29248..: up to 8 frames, delay in tenths of a second).
// ponytail: frames are loaded by the area picture code, not ported yet (M5);
// until then there is never more than frame 0, so nothing animates.

type viewAnim struct {
	frame  int       // b7247: the current frame, 0 = no animation
	frames int       // b7246
	delay  [9]byte   // w7240 + 8i
	pics   [9][]byte // p7244 + 8i
}

// drawAnimationFrame = O110:000A (3, 3, animate, frame): draw the current
// frame into the view window.
func (e *Engine) drawAnimationFrame(animate bool) {}

// mapCursorPlace / Show / Hide = O109:0005 / 002E / 005D: the party's
// blinking cursor on the wilderness map (ported with the wilderness, M5).
func (e *Engine) mapCursorPlace() {}
func (e *Engine) mapCursorShow()  {}
func (e *Engine) mapCursorHide()  {}
