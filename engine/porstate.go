package engine

// Pool of Radiance's interpreter globals still read by the por-derived
// combat, camp and treasure code (to go as those units are ported).

// eclRun holds the ovr04 globals that are not part of the ovr08 interpreter.
type eclRun struct {
	restoreSel  bool       // DS:4430: put DS:4436 back as the selected character
	savedSel    *Character // DS:4436 (a pointer: survives the list changing)
	scriptLoad  byte       // DS:4431
	needFrame   byte       // DS:4432: main screen not drawn yet
	walldefSet  byte       // DS:4433
	geoSet      byte       // DS:4434
	monCount    byte       // DS:4435: monsters appended to the party list
	areaChanged bool       // DS:442F
	quit        bool       // DS:49FF
	searchSave  byte       // DS:84DD
	f84DE       byte
	f84E4       byte
	f84E7       byte // monsters were loaded
	f84D4       byte
	sprite      int        // DS:6F81
	closeup     int        // DS:6F82
	treasure    []*Item    // DS:6810 list
	doorTry     [3]byte    // DS:6DDE..6DE0: bash, pick, knock still allowed
	effTick     [0x49]byte // DS:471D: member i has timed effects to run down
	itemType    byte       // TREASURE: last random item type
	f84DF       byte
}

// cut: s at most n characters.
func cut(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
