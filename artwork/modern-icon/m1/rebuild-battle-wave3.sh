#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建第三批八組怪物三視圖。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

python3 tools/chroma_spritesheet.py \
  --south artwork/modern-icon/m1/masters/battle-wave3-south.png \
  --north artwork/modern-icon/m1/masters/battle-wave3-north.png \
  --east artwork/modern-icon/m1/masters/battle-wave3-east.png \
  --names 06,09,10,17,22,23,24,25 \
  --out artwork/modern-icon/m1/trial
