#!/usr/bin/env bash
# 在 game-video Docker 映像中合成《冬之魔》本機宣傳片。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK="$ROOT/workplace/promo"
mkdir -p "$WORK/sound" "$WORK/out"

# 原版沒有 BGM；只輸出反組譯還原的 PC speaker 九個效果。
"$ROOT/tools/go.sh" run ./cmd/dwsound -out workplace/promo/sound -volume 0.22

docker run --rm --cpus=2 \
    -v "$ROOT:/src:ro" \
    -v "$WORK:/work" \
    -w /src \
    game-video:latest bash tools/promo/render.sh

printf '宣傳片 -> %s\n' "$WORK/out/demon-winter-cht-promo.mp4"
