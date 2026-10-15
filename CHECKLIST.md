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
  - [x] Load / Save game (O0A1 loadGame/SaveGame): savgamX.dat byte-identical but for stack garbage in unused name slots; loading pixel-identical
- [ ] M5 world: 3D view (WALLDEF/8X8D/SKY), area map, ECL interpreter (CotAB opcode set)
  - [x] ECL VM (O04C): operands, packed text, text variables, getVar/setVar with the character values, compares, calls
  - [x] adventuring loop (O03C:3607/3677/3758), command bar, steps, doors (bash/pick/knock), status line, party summary
  - [x] view: adventure frame, EGA sky (bands, sun/moon, SKY 252 floor), 3D view and overhead map, pictures (PIC/SPRIT/HEAD/BODY/bigpic)
  - [x] commands 0..35, 37..64 except COMBAT (36) and PROGRAM (56); first scene .. overhead map pixel-identical to DOSBox
  - [ ] COMBAT (36: shops O048, temple O042, after-combat O045, combat O057) and PROGRAM (56)
  - [ ] the duel (O04C:1B9A), the sound calls (R699, M8)
  - [ ] parity: PARLAY's prompt (the original passes a char as its prompt string)
- [ ] M6 camp, shops, temples, training (paladin/ranger etc.)
  - [x] shop (O048) with money (O0BA: pool, share, take, appraise) and View (O0AF viewPlayer, items menu, trade/drop coins, paladin heal/cure): pixel-identical
  - [x] camp (O09D) and resting (O0B6): menus, magic, memorize, scribe, display, rest, alter, fix: pixel-identical but for campfire frames
  - [x] temple (O042)
  - [ ] item functions still Pool of Radiance's, being audited: ready, use, trade, halve, join, sell, identify
  - [ ] training halls (O0A7 train_player is ported; the hall's script path)
- [ ] M7 combat
  - [x] rules (O0E3): affect groups, saving throw, damage, add/kill/heal/revive, poison and stinking clouds
  - [x] affect handlers (O064 setup_spells2, 148 entries incl. O08D's engulf/hug/rays and the item call)
  - [x] clouds (O0C2 noxious/poisonous cloud, O064 clearing)
  - [x] O05D combat setup: CotAB wilderness map (cliffs, trees, scatter), dungeon map with rubble, placement, blocks
  - [x] O057 combat loop: turn order, camp_menu/combat_menu (extended keys step at once), move/attack, delay menu, speed, quick; battle01 with the 15-round limit (reset by every attack)
  - [x] O08D aiming (manual aim, target picking by hit dice/area, ranged rule), casterLevel (paladin/ranger offsets), god_intervene (debug word "Wooden")
  - [x] O0C2 spells 1-100 (setup_spells) and the breath/gaze/spit handlers it adds (affects 83, 86, 88, 90, 126, 128, 131, 132); cone (17CF), bolt (3986/3A44), hold (1B24), dispel with clouds (31E4)
  - [x] O05A computer turn: every procedure checked; canCast reads the spell table and saves area victims, the
    step check (073B) uses O121:0D04's cloud flags, attack loop bandages first, DS:75FE kept as aiEngaged
  - [x] No START.EXE reads at runtime: the last Pool-address reads (status names, facing names, sky colour,
    item words, class names, line directions) use the generated CotAB tables; Pool-only code removed (area map,
    portrait chooser, endgame fade, Pool RandomItem); Modify Character is CotAB's O0A7:2830 (name editing, HP)
  - [x] O121 combat map: audited and relabelled; death flicker (9 frames, icon slot 0x18 = DS:66B0), redrawChar only while b7600
  - [ ] O08D rest, O11C/O12A map, O045 (remaining Pool stand-ins: o14_*, some o23_* names)
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
- maxHitPoints / setRolledHP (O0A7:4793, modifyPlayer): a character with no class level returns instead of a divide error.
- aiPickWeapon (O05A:1673): ready_Item is not called with a nil weapon when the off hand is empty or cursed.
- Shops with the normal price factor (16): the original's Buy (O048:0474) has no case for it and charges an
  unset local, which holds the menu colour pushed by the shop menu's displayInput (15) — every item costs
  15 gold. The port charges the price the list shows.
- Character sheet movement (O0AF): the original doubles for affect 42 (slow) and halves for 39 (haste), the
  reverse of combat (O064:0BA5/101B, 39 ages and doubles) and encounters (O04C:20E5); the port's sheet
  doubles for haste (39) and halves for slow (42).
- Ice body (O064:1243) tests DS:5D0B (unrelated) where fire body tests the saving throw; the port tests the
  saving throw (DS:6FC2) for both.
- Constitution save bonus (O064:1E42) below 4 adds an unset local; the port adds nothing.
- Appraise "Keep" with no items lost the gem; the port keeps it.
- ENCOUNTER with a digit that is no menu item read past its reaction table; the port asks again.
- Cones (O0C2:17CF) hand lineOfSight an unset local as the range; the port gives no range limit (walls still
  cut the cone).
- Kept as in the original (not bugs to the player): Dispel Magic looks for Cloudkill's squares in Stinking
  Cloud's table run on (DS:27D9); Entangle's duration comes from bytes past the spell table (6 + 3/level);
  once Fear's cone meets a combatant twice, later ones on the line are left out.
- The shop's "Excuse me" text: unchanged; DIVIDE by zero in scripts gives 0 instead of a runtime error.
