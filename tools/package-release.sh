#!/usr/bin/env bash
# 在 Docker 內建立不含原版資料與倚天字型的 Linux／Windows 發行包。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-dev}"
TARGET_GOOS="${2:-linux}"
TARGET_GOARCH="${3:-amd64}"
SAFE_VERSION="${VERSION//[^A-Za-z0-9._-]/-}"
case "$TARGET_GOOS/$TARGET_GOARCH" in
  linux/amd64)
    TARGET_CGO=1
    ;;
  windows/amd64)
    TARGET_CGO=0
    ;;
  darwin/*)
    echo "macOS 的 Ebiten／Metal 建置需要 Apple SDK；請使用原生 macOS CI。" >&2
    exit 2
    ;;
  *)
    echo "不支援的封裝目標：$TARGET_GOOS/$TARGET_GOARCH" >&2
    exit 2
    ;;
esac
NAME="demonwinter-zh-Hant-${SAFE_VERSION}-${TARGET_GOOS}-${TARGET_GOARCH}"

# 建置、staging、授權掃描、壓縮與雜湊全部在同一個一次性容器內完成。
# 主機只負責啟動 Docker；輸出以目前 UID/GID 寫回 dist/。
docker run --rm \
    --network none \
    --memory 2g \
    --cpus 2 \
    --pids-limit 256 \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -e GOCACHE=/gocache \
    -e GOMODCACHE=/gomod \
    -e "RELEASE_NAME=$NAME" \
    -e "TARGET_GOOS=$TARGET_GOOS" \
    -e "TARGET_GOARCH=$TARGET_GOARCH" \
    -e "TARGET_CGO=$TARGET_CGO" \
    -e OUTPUT_DIR=/src/dist \
    -v "$REPO_ROOT:/src" \
    -v dw-gomod:/gomod \
    -v dw-gobuild:/gocache \
    -w /src \
    demonwinter-go bash tools/package-release-inner.sh
