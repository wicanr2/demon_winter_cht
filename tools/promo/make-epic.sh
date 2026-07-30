#!/usr/bin/env bash
# 使用已錄製母帶與核准配樂合成第二支宣傳片。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK="$ROOT/workplace/promo/epic"
VIDEO_IMAGE=demonwinter-video

mkdir -p "$WORK/out" "$WORK/render"

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
    -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$ROOT:/src:ro" \
    -v "$ROOT/workplace/promo/epic:/work" \
    -v "$ROOT/workplace/promo/score:/work/score:ro" \
    -w /src "$VIDEO_IMAGE" bash tools/promo/render-epic.sh

printf '第二支宣傳片 -> %s\n' "$WORK/out/demon-winter-epic-trailer.mp4"
