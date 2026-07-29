# Demon's Winter remake — persistent agent rules

This file is the compact/restart recovery contract for this repository. Read it before making
changes. It complements `CONTEXT.md`; it does not replace evidence in `docs/re/`.

## Recovery order after compact or handoff

1. Read the newest user request.
2. Read [`CONTEXT.md` §7 Worklist](CONTEXT.md#7-worklist狀態的單一真相來源),
   especially the newest dated baseline at §7.0. Older “next step” paragraphs are historical
   unless the newest baseline explicitly revives them.
3. Inspect `git status --short`. Existing changes belong to the user/current task; do not reset
   or discard them.
4. Read the directly relevant spec/research/playtest document. Do not reopen already settled
   reverse-engineering questions merely because an old checklist row was not updated.
5. Before claiming completion, rerun proportional tests and record material visual/playtest
   evidence in Markdown.

## Product goal and fidelity boundary

- Goal: preserve this 1988 classic as a clean, maintainable Traditional Chinese remake that
  later Chinese-speaking players can finish and understand.
- DOS executable behavior is the primary rules oracle. Manuals and walkthroughs are supporting
  evidence and may describe another platform.
- Keep inference labels honest: confirmed, strongly evidenced, hypothesis, or unknown.
- Unknown bytes must round-trip unchanged. Never invent gameplay to make a worklist look complete.
- Modern controls and accessibility changes are allowed only when documented as remake
  differences and when they do not silently change combat/story rules.

## Original data, fonts, and saves

- Treat `workplace/orig/`, original archives, executables, data, and user-provided fonts as
  read-only reference material.
- Player-visible text, help copy, menu labels, and mode-specific command layouts belong in
  language/data JSON, not Go source. Go may retain stable keys, actions, format arguments,
  and layout behavior only. Every feature batch that adds UI must pass `dwstrings uicheck`
  with zero hardcoded Traditional Chinese strings.
- Never commit or package original `.DAT/.DTT/.SHE/.SHP/.PIC/.PIE` assets or Eten
  `STDFONT.15`/`SPCFONT.15`.
- All test saves go to `/tmp` or an explicit test-output directory. Do not overwrite the original
  `PARTY.DAT`, `nSS.DAT`, `ITEMLOCB.DAT`, or maps.
- Release packages must pass the deny-list scan in `tools/package-release.sh`.

## Modern Icon status

- EGA and CGA are preserved original themes.
- Modern Icon is the third optional remake theme. It is not pixel art and must not be produced by
  downscaling concept sheets into the original 32×28 frames. It may use a high-resolution
  presentation layer while retaining the original logical tile indices, anchors, and hitboxes.
- The user approved `modern-ega-concept.png` as the primary direction and
  `modern-ega-m0-terrain-study-b.png` as supporting reference. The runtime-trial, runtime-proof,
  and direct-downscale-failed images are rejected and must not be used as production art.
- Do not call Modern Icon “final” or “completed redraw” until representative in-game screenshots
  and the full production atlas have passed review.
- Art gate: user reviews direction → representative terrain/combat/monster/ship samples →
  contact-sheet/index checks → full atlas production → same-state screenshot acceptance.
- F8 order is EGA → CGA → Modern Icon → EGA. Theme changes must not mutate save, RNG, collision,
  secret-door visibility, combat, or story state.
- The Eten 16×15 bold font remains shared by all themes.
- Modern Icon 的當前逐項生產順序、完成證據與剩餘工作只認
  [`CONTEXT.md` §7.0](CONTEXT.md#70-一句話現況)；每完成一批 terrain／character／
  combat／monster／ship 素材，都必須先更新該 worklist，再提交程式與畫面證據。

## Debug and A6 verification

- Named debug bookmarks use `-scene`; they position the party but must not silently solve riddles
  or set plot flags. Explicit flags such as `-glyphs` remain separate.
- A6 acceptance is sampled: normal-player early vertical slice plus late/high-risk cases. Do not
  claim a full room-by-room replay unless it was actually performed.
- Use fixed seeds and `tools/playthrough.sh` traces for repeatability. Visual output must be
  captured and inspected; compilation alone is not a visual test.
- Minimum release gates:
  - `go test ./...` under Xvfb for Ebiten packages;
  - `dwstrings check` = 500/500;
  - `dwstrings uicheck` passes;
  - A6 representative encounter plus relevant high-risk smoke tests;
  - `git diff --check`;
  - unpacked release binary smoke test.

## Docker and temporary-resource hygiene

- Use `docker run --rm` for one-shot jobs. Named/background containers require an explicit trap
  and bounded lifetime.
- At the end of every Docker batch, inspect project-related running/stopped containers and remove
  only confirmed leftovers from that batch.
- Delete task-created temporary images after they are no longer needed. Keep a documented,
  reproducible current toolchain image (currently `demonwinter-go`) only when subsequent builds
  or tests still depend on it; do not globally prune other projects.
- Named Go cache volumes `dw-gomod` and `dw-gobuild` are intentional reusable caches. Remove them
  only when the user asks for space cleanup or they are proven corrupt/obsolete.
- Temporary screenshots, extracted release trees, traces, and copied saves belong in `/tmp`;
  clean them after verification when they are no longer evidence.
- Never run broad `docker system prune`, `docker image prune -a`, or delete resources selected
  only by age. Resolve exact project-owned IDs/names first.

## Documentation and release truthfulness

- `README.md` is the public GitHub index and must link to design/research/playtest detail rather
  than duplicate every note.
- `CONTEXT.md` §7 is the internal worklist source of truth.
- Every release statement must distinguish:
  1. original EGA/CGA restoration,
  2. optional Modern Icon remake theme,
  3. generated concept/direction art that is not a runtime atlas.
- Packaging is last: rules, visuals, tests, and A6 sampling precede the release archive.
