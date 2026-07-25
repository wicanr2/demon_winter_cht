#!/usr/bin/env bash
# 在 docker 內跑 go 指令（本專案一律用 docker 編譯）。
#
#   tools/go.sh test ./...
#   tools/go.sh build ./cmd/demonwinter
#   tools/go.sh vet ./...
#
# module cache 放在具名 volume `dw-gomod`，重跑不必重抓相依。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE=demonwinter-go
CACHE_VOL=dw-gomod
BUILD_VOL=dw-gobuild

if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
    echo "[go.sh] 首次執行，建置 $IMAGE …" >&2
    docker build -q -t "$IMAGE" "$REPO_ROOT/docker/go" >&2
fi

# 新建的 volume 屬 root，但下面以呼叫者 uid 執行 → 先把擁有者換過來。
if ! docker volume inspect "$CACHE_VOL" >/dev/null 2>&1 \
   || ! docker volume inspect "$BUILD_VOL" >/dev/null 2>&1; then
    echo "[go.sh] 初始化 cache volume …" >&2
    docker run --rm -v "$CACHE_VOL:/gomod" -v "$BUILD_VOL:/gocache" "$IMAGE" \
        chown -R "$(id -u):$(id -g)" /gomod /gocache
fi

# 以呼叫者的 uid 執行，產出的檔案才不會變成 root 所有。
# 因此 cache 路徑要挑 uid 寫得進去的地方，不能用 /root/.cache。
exec docker run --rm \
    -v "$REPO_ROOT:/src" \
    -v "$CACHE_VOL:/gomod" \
    -v "$BUILD_VOL:/gocache" \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -e GOCACHE=/gocache \
    -e GOMODCACHE=/gomod \
    -w /src \
    "$IMAGE" go "$@"
