#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建第二批八組怪物三視圖。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

python3 tools/chroma_spritesheet.py \
  --south artwork/modern-icon/m1/masters/battle-wave2-south.png \
  --north artwork/modern-icon/m1/masters/battle-wave2-north.png \
  --east artwork/modern-icon/m1/masters/battle-wave2-east.png \
  --names 13,14,15,16,18,19,20,21 \
  --out artwork/modern-icon/m1/trial
