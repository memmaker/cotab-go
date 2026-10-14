# cotab-go — Curse of the Azure Bonds on Go/Ebiten

Source of truth: the byte-perfect Turbo Pascal reconstruction (github.com/memmaker/cotab-re).
Infrastructure (EGA screen, DAX reader, Ebiten host) from por-go; game logic ported per procedure
from the Pascal. Rules: ~/notes/c-to-go.md and goplay INTEGRATION.md (no exe reads at runtime,
no hex/magic numbers, readable names).

- [x] M1 `-import` via internal/gameimport (folders, zip, ISO/BIN/CUE, GOG installer, loose files); COVER.PNG
- [ ] M2 `tables` package: every data table the engine reads, named, generated from START.EXE by `cmd/tablegen` (go:generate); no `e.X` at runtime
- [x] M3 boot: title screen, main menu (START main, title_screen, startGameMenu) — party menu pixel-identical to DOSBox
- [ ] M4 characters: create, load/save, party screens
  - [x] creation (race..alignment, stat roll, silent training, icon editor, save) — every screen pixel-identical, .GUY byte-identical (pointer bytes aside), seeded RNG
  - [x] Add character (Curse .GUY files) with the party rules
  - [ ] Pool of Radiance / Hillsfar import (O0A1 load_player)
  - [ ] Drop / Modify / View / Remove / Human Change Classes screens (ported, not yet parity-checked)
  - [ ] Load / Save game (O0A1 loadGame/SaveGame)
- [ ] M5 world: 3D view (WALLDEF/8X8D/SKY), area map, ECL interpreter (CotAB opcode set)
  - [x] ECL VM (O04C): operands, packed text, text variables, getVar/setVar with the character values, compares, calls
  - [x] adventuring loop (O03C:3607/3677/3758), command bar, steps, doors (bash/pick/knock), status line, party summary
  - [x] view: adventure frame, EGA sky (bands, sun/moon, SKY 252 floor), 3D view and overhead map, pictures (PIC/SPRIT/HEAD/BODY/bigpic)
  - [x] commands 0..35, 37..64 except COMBAT (36) and PROGRAM (56); first scene .. overhead map pixel-identical to DOSBox
  - [ ] COMBAT (36: shops O048, temple O042, after-combat O045, combat O057) and PROGRAM (56)
  - [ ] the duel (O04C:1B9A), the sound calls (R699, M8)
  - [ ] parity: PARLAY's prompt (the original passes a char as its prompt string)
- [ ] M6 camp, shops, temples, training (paladin/ranger etc.)
- [ ] M7 combat
- [ ] M8 sound: the PC speaker driver as Go data
- [ ] M9 goplay: games.js, deploy.sh, README tables, hardware cursor n/a (no pointer), wasm build, box art
- [ ] M10 publish (GitHub + deploy)

## Parity method
`tools/dosparity.py` drives the original in the DOSBox-X remote-debug fork (added QMP `vgadump`,
`memwrite`; `status` reports CS:IP); `cmd/snap game - KEYS` drives the port with the same keys;
`COTAB_SEED` / `seed:N` set Turbo Pascal's RandSeed (DS 18416) on both sides. Screens are compared as
raw EGA colour indices.

## Known placeholders
- viewanim.go: the view-window animation and the wilderness map cursor (M5).
- Camp (O09D) still runs Pool of Radiance's Encamp; combat, shops and temples are por code paths.
- Pool/Hillsfar import shows a message.

## Deliberate deviations (obvious bugs fixed, per c-to-go.md)
- Shops with the normal price factor (16): the original's Buy (O048:0474) has no case for it and charges an
  unset local, which holds the menu colour pushed by the shop menu's displayInput (15) — every item costs
  15 gold. The port charges the price the list shows.
- Encounter speeds (O04C:20E5): haste and slow are swapped against the character sheet; the port doubles for
  haste (42) and halves for slow (39) like the sheet.
- Appraise "Keep" with no items lost the gem; the port keeps it.
- ENCOUNTER with a digit that is no menu item read past its reaction table; the port asks again.
- The shop's "Excuse me" text: unchanged; DIVIDE by zero in scripts gives 0 instead of a runtime error.
