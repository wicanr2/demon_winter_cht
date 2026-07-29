#!/usr/bin/env bash
# 在 demonwinter-go 容器內重建草原海岸組與第二海面。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

go run ./tools/moderncoast \
  -orig workplace/orig/demwin/DEM_DATA \
  -normal-land artwork/modern-icon/m1/trial/normal-plain.png \
  -normal-water artwork/modern-icon/m1/trial/normal-ocean.png \
  -winter-land artwork/modern-icon/m1/trial/winter-plain.png \
  -winter-water artwork/modern-icon/m1/trial/winter-ocean.png \
  -out artwork/modern-icon/m1/trial \
  -tiles 17,1a,1d,20,3b,3c,3d,3e

for material in normal-desert-ground winter-desert-ground \
  normal-forest-ground winter-forest-ground; do
  convert "artwork/modern-icon/m1/masters/$material.png" \
    -filter Lanczos -resize 64x56\! -alpha off \
    "artwork/modern-icon/m1/trial/$material.png"
done

go run ./tools/moderncoast \
  -orig workplace/orig/demwin/DEM_DATA \
  -normal-land artwork/modern-icon/m1/trial/normal-desert-ground.png \
  -normal-water artwork/modern-icon/m1/trial/normal-ocean.png \
  -winter-land artwork/modern-icon/m1/trial/winter-desert-ground.png \
  -winter-water artwork/modern-icon/m1/trial/winter-ocean.png \
  -out artwork/modern-icon/m1/trial \
  -tiles 43,44,45,46,47,48,49,4a

go run ./tools/moderncoast \
  -orig workplace/orig/demwin/DEM_DATA \
  -normal-land artwork/modern-icon/m1/trial/normal-forest-ground.png \
  -normal-water artwork/modern-icon/m1/trial/normal-ocean.png \
  -winter-land artwork/modern-icon/m1/trial/winter-forest-ground.png \
  -winter-water artwork/modern-icon/m1/trial/winter-ocean.png \
  -out artwork/modern-icon/m1/trial \
  -tiles 4b,4c,4d,4e,4f,50,51,52

convert artwork/modern-icon/m1/masters/normal-ocean-62.png \
  -filter Lanczos -resize 64x56\! -alpha off /tmp/normal-ocean-62.png
convert artwork/modern-icon/m1/masters/winter-ocean-62.png \
  -filter Lanczos -resize 64x56\! -alpha off /tmp/winter-ocean-62.png
convert -size 64x56 xc:black \
  -fx 'min(1,max(0,min(min(i,j),min(w-1-i,h-1-j))/8))' /tmp/coast-edge-mask.png

# 中央使用獨立浪紋，邊緣漸變回 0x14，確保任意相鄰排列不產生硬縫。
convert artwork/modern-icon/m1/trial/normal-ocean.png \
  /tmp/normal-ocean-62.png /tmp/coast-edge-mask.png \
  -compose over -composite artwork/modern-icon/m1/trial/normal-ocean-62.png
convert artwork/modern-icon/m1/trial/winter-ocean.png \
  /tmp/winter-ocean-62.png /tmp/coast-edge-mask.png \
  -compose over -composite artwork/modern-icon/m1/trial/winter-ocean-62.png
