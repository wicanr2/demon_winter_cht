#!/usr/bin/env bash
# 第二支 72 秒史詩宣傳片的實機母帶。全部由 playthrough/X11 真實錄製。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="$ROOT/workplace/promo/epic/capture"
mkdir -p "$OUT"

record() {
    local name=$1 script=$2
    shift 2
    DW_RECORD="$name.mp4" DW_RECORD_FPS=25 \
        "$ROOT/tools/playthrough.sh" "$ROOT/tools/promo/$script" "$OUT/$name" \
        -seed=11 -eten workplace/eten -eten-bold=true -controls=modern \
        -music-volume=0 -volume=0 "$@"
}

record f8 epic-f8.txt -video=ega -map=34 -x=28 -y=50
record dungeon epic-dungeon.txt -video=modern -scene=armory
record town epic-town.txt -video=modern -town=1
record boss epic-boss.txt -video=modern -battle -autofight \
    -battle-monsters=41,74,91,10
record sea epic-sea.txt -video=modern -sea-battle -autofight

printf '第二支宣傳片實機母帶 -> %s\n' "$OUT"
