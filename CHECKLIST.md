# cotab-go — Curse of the Azure Bonds on Go/Ebiten

Source of truth: the byte-perfect Turbo Pascal reconstruction (github.com/memmaker/cotab-re).
Infrastructure (EGA screen, DAX reader, Ebiten host) from por-go; game logic ported per procedure
from the Pascal. Rules: ~/notes/c-to-go.md and goplay INTEGRATION.md (no exe reads at runtime,
no hex/magic numbers, readable names).

- [x] M1 `-import` via internal/gameimport (folders, zip, ISO/BIN/CUE, GOG installer, loose files); COVER.PNG
- [ ] M2 `tables` package: every data table the engine reads, named, generated from START.EXE by `cmd/tablegen` (go:generate); no `e.X` at runtime
- [ ] M3 boot: title screen, main menu (START main, title_screen, startGameMenu)
- [ ] M4 characters: create, load/save (CHRDAT/SAVGAM formats), party screens
- [ ] M5 world: 3D view (WALLDEF/8X8D/SKY), area map, ECL interpreter (CotAB opcode set)
- [ ] M6 camp, shops, temples, training (paladin/ranger etc.)
- [ ] M7 combat
- [ ] M8 sound: the PC speaker driver as Go data
- [ ] M9 goplay: games.js, deploy.sh, README tables, hardware cursor n/a (no pointer), wasm build, box art
- [ ] M10 publish (GitHub + deploy)
