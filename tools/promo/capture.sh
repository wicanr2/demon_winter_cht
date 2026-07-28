#!/usr/bin/env bash
# 擷取宣傳片用的真實遊戲操作。產物含 MP4、trace 與關鍵 PNG，全部在
# gitignored 的 workplace/promo/capture。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="$ROOT/workplace/promo/capture"
mkdir -p "$OUT"

DW_RECORD=world.mp4 DW_RECORD_FPS=25 \
    "$ROOT/tools/playthrough.sh" "$ROOT/tools/promo/world.txt" "$OUT/world" \
    -seed=11 -autofight -eten workplace/eten -eten-bold=true

DW_RECORD=battle.mp4 DW_RECORD_FPS=25 \
    "$ROOT/tools/playthrough.sh" "$ROOT/tools/promo/battle.txt" "$OUT/battle" \
    -seed=11 -battle -autofight -battle-monsters=13,14,15,16,17 \
    -eten workplace/eten -eten-bold=true

printf '實機片段 -> %s\n' "$OUT"
