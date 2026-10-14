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
- Pool/Hillsfar import shows a message.
