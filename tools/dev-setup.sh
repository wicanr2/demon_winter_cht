#!/usr/bin/env bash
# 從 repo + 私人輸入建立並驗證 Demon's Winter remake 開發環境。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
IMAGE=demonwinter-go

step() { printf '\n==> %s\n' "$*"; }
fail() { printf '!! %s\n' "$*" >&2; exit 1; }

command -v docker >/dev/null 2>&1 || fail "需要 Docker；見 docs/DEV_SETUP.md"
case "${1:-}" in
    ""|--rebuild-only) ;;
    *) fail "用法：bash tools/dev-setup.sh [--rebuild-only]" ;;
esac

if [ "${1:-}" != "--rebuild-only" ]; then
    step "1/6 建置 Docker 工具鏈 $IMAGE"
    docker build --memory 2g -t "$IMAGE" -f docker/go/Dockerfile docker/go
else
    step "1/6 使用既有 Docker 工具鏈 $IMAGE"
    docker image inspect "$IMAGE" >/dev/null 2>&1 ||
        fail "找不到 $IMAGE；請不要使用 --rebuild-only"
fi

run_go() {
    docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
        -u "$(id -u):$(id -g)" -e HOME=/tmp \
        -e GOCACHE=/gocache -e GOMODCACHE=/gomod \
        -v "$ROOT:/src" -v dw-gomod:/gomod -v dw-gobuild:/gocache \
        -w /src "$IMAGE" "$@"
}

step "2/6 私人 oracle、94 個資料檔與倚天字型"
run_go bash -c '
set -e
for required in \
    workplace/orig/demwin/DEMON.EXE \
    workplace/orig/demwin/DEMON.INT \
    workplace/orig/demwin/DEM_DATA/FILES.DAT \
    workplace/orig/demwin/DEM_DATA/FILES.DTT \
    workplace/orig/demwin/DEM_DATA/SUM.MAP \
    workplace/orig/demwin/DEM_DATA/MONSTER.DAT \
    workplace/eten/STDFONT.15 \
    workplace/eten/SPCFONT.15; do
    test -s "$required" || { echo "缺少私人輸入：$required" >&2; exit 1; }
done
count=$(find workplace/orig/demwin/DEM_DATA -maxdepth 1 -type f | wc -l)
test "$count" -eq 94 || {
    echo "DEM_DATA 應為 94 個檔案，現在是 $count" >&2
    exit 1
}'

step "3/6 Go／Ebiten 全套測試"
run_go bash -c '
set -e
Xvfb :99 -screen 0 1600x900x24 >/tmp/xvfb.log 2>&1 &
xvfb_pid=$!
trap "kill -9 $xvfb_pid 2>/dev/null || true" EXIT INT TERM
for i in $(seq 1 50); do
    xdpyinfo -display :99 >/dev/null 2>&1 && break
    sleep 0.1
done
DISPLAY=:99 go test ./...'

step "4/6 原版資料文字 500/500"
run_go go run ./cmd/dwstrings check \
    -data workplace/orig/demwin/DEM_DATA -lang assets/lang/zh-Hant

step "5/6 玩家介面 JSON 與硬編字串"
run_go go run ./cmd/dwstrings uicheck \
    -src cmd/demonwinter -rules internal/game -lang assets/lang/zh-Hant

step "6/6 工作樹格式與 Docker 清理狀態"
git diff --check
leftovers="$(
    docker ps -a --filter 'name=dw-' --filter 'name=demonwinter' \
        --format '{{.Names}}'
)"
test -z "$leftovers" || fail "發現專案容器殘留：$leftovers"

printf '\n開發環境驗證完成。接續閱讀 docs/DEV_SETUP.md 與 CONTEXT.md §7。\n'
