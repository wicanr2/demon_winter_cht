#!/usr/bin/env bash
# 建立不含原版資料與倚天字型的 Linux amd64 發行包。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-dev}"
SAFE_VERSION="${VERSION//[^A-Za-z0-9._-]/-}"
DIST="$REPO_ROOT/dist"
mkdir -p "$DIST"
# tools/go.sh 只把 repo 掛進 build container；staging 必須位於 repo 內，
# 否則 `go build -o /tmp/...` 只會寫到短命容器的 /tmp，主機壓縮時看不到 binary。
STAGE="$(mktemp -d "$DIST/.release-stage-XXXXXX")"
STAGE_REL="${STAGE#"$REPO_ROOT/"}"
NAME="demonwinter-zh-Hant-${SAFE_VERSION}-linux-amd64"
PACKAGE="$DIST/$NAME.tar.gz"

cleanup() { rm -rf "$STAGE"; }
trap cleanup EXIT

mkdir -p "$STAGE/$NAME/assets/lang/zh-Hant" "$STAGE/$NAME/assets/manual/zh-Hant"
mkdir -p "$STAGE/$NAME/artwork/modern-icon/m1/trial"

"$REPO_ROOT/tools/go.sh" build -trimpath -ldflags="-s -w" \
    -o "$STAGE_REL/$NAME/demonwinter" ./cmd/demonwinter
if [[ ! -x "$STAGE/$NAME/demonwinter" ]]; then
    echo "拒絕打包：引擎執行檔不存在或不可執行" >&2
    exit 1
fi

cp "$REPO_ROOT/README.md" "$STAGE/$NAME/README.md"
cp "$REPO_ROOT/packaging/README-zh-Hant.txt" "$STAGE/$NAME/開始遊戲.txt"
cp "$REPO_ROOT/assets/lang/zh-Hant/"* "$STAGE/$NAME/assets/lang/zh-Hant/"
cp "$REPO_ROOT/assets/manual/zh-Hant/"* "$STAGE/$NAME/assets/manual/zh-Hant/"
# Modern Icon 是本專案自製的第三主題，不含原版素材；保留與開發樹相同的
# 相對路徑，讓執行檔不加參數也能載入，F8 才不是只切到舊調色預覽。
cp "$REPO_ROOT/artwork/modern-icon/m1/trial/"* \
    "$STAGE/$NAME/artwork/modern-icon/m1/trial/"

# 授權邊界是發行契約，不能只靠「大家應該知道」。若 staging 意外出現
# 原版副檔名或倚天檔名，直接拒絕打包。
if find "$STAGE/$NAME" -type f \
    \( -iname '*.DAT' -o -iname '*.DTT' -o -iname '*.SHE' -o -iname '*.SHP' \
       -o -iname '*.PIE' -o -iname '*.PIC' -o -iname 'STDFONT.15' \
       -o -iname 'SPCFONT.15' \) | grep -q .; then
    echo "拒絕打包：staging 內含原版資料或倚天字型" >&2
    exit 1
fi

tar -C "$STAGE" -czf "$PACKAGE" "$NAME"
sha256sum "$PACKAGE" > "$PACKAGE.sha256"

printf '發行包：%s\n' "$PACKAGE"
printf '校驗碼：%s.sha256\n' "$PACKAGE"
