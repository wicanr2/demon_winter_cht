#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建 Modern Icon 地城 D2–D4 索引。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

tmp_dir="$(mktemp -d /tmp/dw-dungeon-d2-d4-XXXXXX)"
cleanup() { rm -rf "$tmp_dir"; }
trap cleanup EXIT INT TERM

prepare() {
  local source="$1" out="$2"
  convert "$source" -resize '900x900^' -gravity center -extent 900x900 "$out"
}

prepare artwork/modern-icon/m1/masters/dungeon-carved-face.png "$tmp_dir/face-12.png"
prepare artwork/modern-icon/m1/masters/dungeon-stairs.png "$tmp_dir/stairs-24.png"
convert "$tmp_dir/stairs-24.png" -rotate 180 "$tmp_dir/stairs-5c.png"
prepare artwork/modern-icon/m1/masters/dungeon-wood-door.png "$tmp_dir/door-2c.png"
prepare artwork/modern-icon/m1/masters/dungeon-portcullis.png "$tmp_dir/gate-39.png"
convert "$tmp_dir/gate-39.png" -flop -modulate 96,90,105 "$tmp_dir/gate-3a.png"
prepare artwork/modern-icon/m1/masters/dungeon-brazier.png "$tmp_dir/brazier-32.png"
prepare artwork/modern-icon/m1/masters/dungeon-cache.png "$tmp_dir/cache-53.png"
prepare artwork/modern-icon/m1/masters/dungeon-ice-shrine.png "$tmp_dir/ice-shrine-41.png"
prepare artwork/modern-icon/m1/masters/dungeon-ice-doorway.png "$tmp_dir/ice-door-37.png"
convert "$tmp_dir/ice-door-37.png" -flop "$tmp_dir/ice-door-54.png"
convert "$tmp_dir/ice-door-37.png" -modulate 88,90,105 "$tmp_dir/ice-door-5d.png"

prepare artwork/modern-icon/m1/masters/dungeon-ice-wall.png "$tmp_dir/ice-wall.png"
convert "$tmp_dir/ice-wall.png" -crop 520x460+30+80 +repage "$tmp_dir/ice-wall-57.png"
convert "$tmp_dir/ice-wall.png" -crop 520x460+350+360 +repage "$tmp_dir/ice-wall-59.png"

prepare artwork/modern-icon/m1/masters/dungeon-rubble.png "$tmp_dir/rubble.png"
convert "$tmp_dir/rubble.png" -crop 520x460+180+210 +repage "$tmp_dir/rubble-58.png"

# 0x31 是另一區的大面積阻擋牆；保留牆拓撲，但去除 0x0d 的緋紅色，
# 讓兩座地城在視覺上可辨。
convert artwork/modern-icon/m1/trial/dungeon-crimson-wall-0d.png \
  -modulate 82,18,100 "$tmp_dir/stone-wall-31.png"

python3 tools/seamless_set.py --width 64 --height 56 --blend 8 \
  "$tmp_dir/face-12.png:artwork/modern-icon/m1/trial/dungeon-carved-face-12.png" \
  "$tmp_dir/stairs-24.png:artwork/modern-icon/m1/trial/dungeon-stairs-24.png" \
  "$tmp_dir/stairs-5c.png:artwork/modern-icon/m1/trial/dungeon-stairs-5c.png" \
  "$tmp_dir/door-2c.png:artwork/modern-icon/m1/trial/dungeon-door-2c.png" \
  "$tmp_dir/stone-wall-31.png:artwork/modern-icon/m1/trial/dungeon-stone-wall-31.png" \
  "$tmp_dir/brazier-32.png:artwork/modern-icon/m1/trial/dungeon-brazier-32.png" \
  "$tmp_dir/ice-door-37.png:artwork/modern-icon/m1/trial/dungeon-ice-door-37.png" \
  "$tmp_dir/gate-39.png:artwork/modern-icon/m1/trial/dungeon-portcullis-39.png" \
  "$tmp_dir/gate-3a.png:artwork/modern-icon/m1/trial/dungeon-portcullis-3a.png" \
  "$tmp_dir/ice-shrine-41.png:artwork/modern-icon/m1/trial/dungeon-ice-shrine-41.png" \
  "$tmp_dir/cache-53.png:artwork/modern-icon/m1/trial/dungeon-cache-53.png" \
  "$tmp_dir/ice-door-54.png:artwork/modern-icon/m1/trial/dungeon-ice-door-54.png" \
  "$tmp_dir/ice-wall-57.png:artwork/modern-icon/m1/trial/dungeon-ice-wall-57.png" \
  "$tmp_dir/rubble-58.png:artwork/modern-icon/m1/trial/dungeon-rubble-58.png" \
  "$tmp_dir/ice-wall-59.png:artwork/modern-icon/m1/trial/dungeon-ice-wall-59.png" \
  "$tmp_dir/ice-door-5d.png:artwork/modern-icon/m1/trial/dungeon-ice-door-5d.png"

# 0x5e–0x61 是四向牆角。以已核准牆／地板直接合成，保留原版三角拓撲；
# 不從概念稿或原版圖塊裁切。
floor=artwork/modern-icon/m1/trial/dungeon-slate-floor-00.png
wall=artwork/modern-icon/m1/trial/dungeon-crimson-wall-0d.png
for spec in \
  "5e:0,0 64,0 0,56" \
  "5f:0,0 64,0 64,56" \
  "60:0,0 0,56 64,56" \
  "61:64,0 0,56 64,56"; do
  index="${spec%%:*}"
  points="${spec#*:}"
  mask="$tmp_dir/mask-$index.png"
  convert -size 64x56 xc:black -fill white -draw "polygon $points" "$mask"
  convert "$floor" "$wall" "$mask" -composite \
    "artwork/modern-icon/m1/trial/dungeon-corner-$index.png"
done
