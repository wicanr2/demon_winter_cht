#!/usr/bin/env bash
# 產生 P4 最終視覺審查的 5 類場景 × 3 主題同狀態截圖。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-$ROOT/docs/design/img/p4}"

if [[ "${P4_IN_CONTAINER:-}" != 1 ]]; then
  mkdir -p "$OUT"
  OUT_ABS="$(cd "$OUT" && pwd)"
  exec docker run --rm \
    --network none \
    --memory 2g \
    --cpus 2 \
    --pids-limit 256 \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -e GOCACHE=/gocache \
    -e GOMODCACHE=/gomod \
    -e LIBGL_ALWAYS_SOFTWARE=1 \
    -e P4_IN_CONTAINER=1 \
    -v "$ROOT:/src" \
    -v "$OUT_ABS:/out" \
    -v dw-gomod:/gomod \
    -v dw-gobuild:/gocache \
    -w /src \
    demonwinter-go bash tools/p4-capture.sh /out
fi

go build -o /tmp/demonwinter ./cmd/demonwinter
Xvfb :99 -screen 0 1600x900x24 >/tmp/p4-xvfb.log 2>&1 &
xvfb_pid=$!
cleanup() {
  jobs -pr | xargs -r kill -9 2>/dev/null || true
  kill -9 "$xvfb_pid" 2>/dev/null || true
}
trap cleanup EXIT INT TERM
for _ in $(seq 1 50); do
  xdpyinfo -display :99 >/dev/null 2>&1 && break
  sleep 0.1
done
export DISPLAY=:99

capture() {
  local kind="$1" theme="$2" keys="$3"
  shift 3
  local save="/tmp/p4-${kind}-${theme}.DAT"
  /tmp/demonwinter \
    -data workplace/orig/demwin/DEM_DATA \
    -eten workplace/eten -eten-bold=true \
    -modern-icon-dir artwork/modern-icon/m1/trial \
    -seed=11 -volume=0 -save "$save" -video "$theme" "$@" \
    >"/tmp/p4-${kind}-${theme}.log" 2>&1 &
  local app_pid=$!
  local wid=""
  for _ in $(seq 1 50); do
    wid="$(xdotool search --onlyvisible --name '冬之魔' 2>/dev/null | tail -1 || true)"
    [[ -n "$wid" ]] && break
    sleep 0.1
  done
  if [[ -z "$wid" ]]; then
    cat "/tmp/p4-${kind}-${theme}.log" >&2
    return 1
  fi
  xdotool windowactivate --sync "$wid" 2>/dev/null || true
  for key in ${keys//,/ }; do
    xdotool key --window "$wid" --clearmodifiers "$key"
    sleep 0.25
  done
  sleep 0.5
  import -window root "$OUT/p4-${kind}-${theme}.png"
  kill -9 "$app_pid" 2>/dev/null || true
  wait "$app_pid" 2>/dev/null || true
}

for theme in ega cga modern; do
  capture world "$theme" "Return,Up" -map=34 -x=28 -y=50
  capture winter "$theme" "Return,Up,Tab" -map=34 -x=28 -y=50
  capture dungeon "$theme" "Return" -scene=armory
  capture battle "$theme" "Return" -battle -battle-monsters=13,14,15,16,17
  capture sea "$theme" "Return" -sea-battle
done

printf 'P4 screenshots: %s\n' "$OUT"
