package engine

// Human dual-classing (unit O0FE of the reconstruction).

// raceHuman is the Race value of humans.
const raceHuman = 7

// noClass: activeClass found no class in progress.
const noClass = 17

// activeClass = O0FE:12AA: for a human, the first class with levels,
// if that class is still active (positive level); noClass otherwise and
// for every other race.
func activeClass(c *Character) int {
	if c.Race != raceHuman {
		return noClass
	}
	k := 0
	for k < clsMonk && c.OldLevels[k] == 0 {
		k++
	}
	if int8(c.OldLevels[k]) > 0 {
		return k
	}
	return noClass
}

// mayChangeClass tells whether the party menu offers "Human Change Classes"
// for c: a human who has not changed class yet (O0FE:13B2 and 12AA).
func (e *Engine) mayChangeClass(c *Character) bool {
	return c != nil && c.Race == raceHuman && activeClass(c) == noClass
}

// NPC tells whether c is a non-player character (control value above 127).
func (c *Character) NPC() bool { return c.Control >= 128 }

// gameFinished: the end of the game was reached (area word 1018, set to 255
// by the ending): the party may not begin adventuring again.
func (e *Engine) gameFinished() bool { return wordAt(e.area, gameFinishedWord) != 0 }

const gameFinishedWord = 1018
