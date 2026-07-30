#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建 Modern Icon 地城 D1 基底素材。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

tmp_dir="$(mktemp -d /tmp/dw-dungeon-d1-XXXXXX)"
cleanup() { rm -rf "$tmp_dir"; }
trap cleanup EXIT INT TERM

slate="artwork/modern-icon/m1/masters/dungeon-slate-floor-d1.png"
crimson="artwork/modern-icon/m1/masters/dungeon-crimson-floor-13.png"
wall="artwork/modern-icon/m1/masters/dungeon-crimson-wall-0d.png"

# 0x00 與 0x56 在原版 EGA/CGA 都是純黑可走地面；Modern Icon 以同一種
# 核准石板材質的不同區段重畫，保留兩個索引，不把規則資料合併。
convert "$slate" -resize '1200x1200^' -gravity center -extent 1200x1200 \
  "$tmp_dir/slate.png"
convert "$tmp_dir/slate.png" -crop 520x460+80+100 +repage \
  "$tmp_dir/slate-00.png"
convert "$tmp_dir/slate.png" -crop 520x460+600+630 +repage \
  "$tmp_dir/slate-56.png"

# 0x13 是原版緋紅磚面；另用核准的緋紅材質，不從方向稿裁切。
convert "$crimson" -resize '900x900^' -gravity center -extent 900x900 \
  "$tmp_dir/crimson-13.png"
convert "$wall" -resize '900x900^' -gravity center -extent 900x900 \
  "$tmp_dir/wall-0d.png"

python3 tools/seamless_set.py --width 64 --height 56 --blend 8 \
  "$tmp_dir/slate-00.png:artwork/modern-icon/m1/trial/dungeon-slate-floor-00.png" \
  "$tmp_dir/slate-56.png:artwork/modern-icon/m1/trial/dungeon-slate-floor-56.png" \
  "$tmp_dir/crimson-13.png:artwork/modern-icon/m1/trial/dungeon-crimson-floor-13.png" \
  "$tmp_dir/wall-0d.png:artwork/modern-icon/m1/trial/dungeon-crimson-wall-0d.png"
