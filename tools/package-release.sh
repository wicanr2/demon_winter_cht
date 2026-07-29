#!/usr/bin/env bash
# 建立不含原版資料與倚天字型的 Linux amd64 發行包。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-dev}"
SAFE_VERSION="${VERSION//[^A-Za-z0-9._-]/-}"
NAME="demonwinter-zh-Hant-${SAFE_VERSION}-linux-amd64"

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
    -v "$REPO_ROOT:/src" \
    -v dw-gomod:/gomod \
    -v dw-gobuild:/gocache \
    -w /src \
    demonwinter-go bash -c '
set -euo pipefail

mkdir -p /src/dist
stage="$(mktemp -d /src/dist/.release-stage-XXXXXX)"
cleanup() { rm -rf "$stage"; }
trap cleanup EXIT INT TERM

root="$stage/$RELEASE_NAME"
mkdir -p "$root/assets/lang/zh-Hant" "$root/assets/manual/zh-Hant"
mkdir -p "$root/artwork/modern-icon/m1/trial"

/usr/local/go/bin/go build -trimpath -ldflags="-s -w" \
    -o "$root/demonwinter" ./cmd/demonwinter
if [[ ! -x "$root/demonwinter" ]]; then
    echo "拒絕打包：引擎執行檔不存在或不可執行" >&2
    exit 1
fi

cp README.md "$root/README.md"
cp packaging/README-zh-Hant.txt "$root/開始遊戲.txt"
cp assets/lang/zh-Hant/* "$root/assets/lang/zh-Hant/"
cp assets/manual/zh-Hant/* "$root/assets/manual/zh-Hant/"
# Modern Icon 是本專案自製的第三主題，不含原版素材；保留開發樹相對路徑。
cp artwork/modern-icon/m1/trial/* "$root/artwork/modern-icon/m1/trial/"

# 授權邊界是發行契約；命中任何原版資料或倚天字型就直接拒絕打包。
if find "$root" -type f \
    \( -iname "*.DAT" -o -iname "*.DTT" -o -iname "*.SHE" -o -iname "*.SHP" \
       -o -iname "*.PIE" -o -iname "*.PIC" -o -iname "STDFONT.15" \
       -o -iname "SPCFONT.15" \) | grep -q .; then
    echo "拒絕打包：staging 內含原版資料或倚天字型" >&2
    exit 1
fi

package="/src/dist/$RELEASE_NAME.tar.gz"
tar -C "$stage" -czf "$package" "$RELEASE_NAME"
(cd /src/dist && sha256sum "$RELEASE_NAME.tar.gz" > "$RELEASE_NAME.tar.gz.sha256")
printf "發行包：%s\n" "$package"
printf "校驗碼：%s.sha256\n" "$package"
'
