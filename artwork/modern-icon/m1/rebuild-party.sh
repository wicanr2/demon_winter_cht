#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建世界隊伍四向、兩步的透明 64×56 試片。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

# 西向沿用東向角色與步態的精確鏡射，避免重新生成造成裝備與比例漂移。
for step in a b; do
  convert "artwork/modern-icon/m1/masters/party-east-$step.png" -flop \
    "artwork/modern-icon/m1/masters/party-west-$step.png"
done

for direction in north east south west; do
  for step in a b; do
    convert "artwork/modern-icon/m1/masters/party-$direction-$step.png" \
      -filter Lanczos -resize 64x56\! \
      "artwork/modern-icon/m1/trial/party-$direction-$step.png"
  done
done

# 航海圖示是單一北向母圖的精確旋轉；四格沒有走路相位。
convert "artwork/modern-icon/m1/masters/ship-north.png" -rotate 90 \
  "artwork/modern-icon/m1/masters/ship-east.png"
convert "artwork/modern-icon/m1/masters/ship-north.png" -rotate 180 \
  "artwork/modern-icon/m1/masters/ship-south.png"
convert "artwork/modern-icon/m1/masters/ship-north.png" -rotate -90 \
  "artwork/modern-icon/m1/masters/ship-west.png"
for direction in north east south west; do
  convert "artwork/modern-icon/m1/masters/ship-$direction.png" \
    -filter Lanczos -resize 64x56\! \
    "artwork/modern-icon/m1/trial/ship-$direction.png"
done
