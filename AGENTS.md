# Demon's Winter remake — persistent agent rules

This file is the compact/restart recovery contract for this repository. Read it before making
changes. It complements `CONTEXT.md`; it does not replace evidence in `docs/re/`.

## Recovery order after compact or handoff

1. Read the newest user request.
2. Read `CONTEXT.md` §7, especially the newest dated baseline at §7.0. Older “next step”
   paragraphs are historical unless the newest baseline explicitly revives them.
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
- Never commit or package original `.DAT/.DTT/.SHE/.SHP/.PIC/.PIE` assets or Eten
  `STDFONT.15`/`SPCFONT.15`.
- All test saves go to `/tmp` or an explicit test-output directory. Do not overwrite the original
  `PARTY.DAT`, `nSS.DAT`, `ITEMLOCB.DAT`, or maps.
- Release packages must pass the deny-list scan in `tools/package-release.sh`.

## Modern EGA status

- EGA and CGA are preserved original themes.
- Modern EGA is a third optional remake theme. The current runtime implementation is a complete,
  index-preserving palette preview, not user-approved frame-by-frame replacement art.
- Do not call Modern EGA “final”, “approved”, or “completed redraw” until the user has reviewed
  the direction and representative in-game screenshots.
- Art gate: user reviews direction → representative terrain/combat/monster/ship samples →
  contact-sheet/index checks → full atlas production → same-state screenshot acceptance.
- F8 order is EGA → CGA → Modern EGA → EGA. Theme changes must not mutate save, RNG, collision,
  secret-door visibility, combat, or story state.
- The Eten 16×15 bold font remains shared by all themes.

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
  2. optional Modern EGA remake preview,
  3. generated concept/direction art that is not a runtime atlas.
- Packaging is last: rules, visuals, tests, and A6 sampling precede the release archive.
