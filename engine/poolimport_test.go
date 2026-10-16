package engine

import "testing"

// A paladin or ranger from por-go gets its creation powers on import.
func TestPoolClassPowers(t *testing.T) {
	e := &Engine{}
	p := &Character{}
	p.Levels[clsPaladin] = 8
	e.poolClassPowers(p)
	e.poolClassPowers(p) // twice: still one affect
	if p.PaladinCures != 2 || e.FindAffect(p, affectProtectionFromEvil) == nil || len(p.Affects) != 1 {
		t.Errorf("paladin: cures %d affects %d", p.PaladinCures, len(p.Affects))
	}
	r := &Character{}
	r.Levels[clsRanger] = 7
	e.poolClassPowers(r)
	if e.FindAffect(r, affectRangerVsGiants) == nil || r.PaladinCures != 0 {
		t.Error("ranger")
	}
}
