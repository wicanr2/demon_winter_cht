#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建最終四組怪物三視圖。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

python3 tools/chroma_spritesheet.py \
  --south artwork/modern-icon/m1/masters/battle-wave4-south.png \
  --north artwork/modern-icon/m1/masters/battle-wave4-north.png \
  --east artwork/modern-icon/m1/masters/battle-wave4-east.png \
  --names 26,27,28,29 \
  --out artwork/modern-icon/m1/trial
