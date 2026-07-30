#!/usr/bin/env bash
# 在已具備目標平台 Go 工具鏈的環境建立乾淨發行包。
# Linux／Windows 由 package-release.sh 在 Docker 內呼叫；macOS 由原生 CI 呼叫。
set -euo pipefail

: "${RELEASE_NAME:?缺少 RELEASE_NAME}"
: "${TARGET_GOOS:?缺少 TARGET_GOOS}"
: "${TARGET_GOARCH:?缺少 TARGET_GOARCH}"

OUTPUT_DIR="${OUTPUT_DIR:-$PWD/dist}"
TARGET_CGO="${TARGET_CGO:-0}"
binary="demonwinter"
if [[ "$TARGET_GOOS" == "windows" ]]; then
  binary+=".exe"
fi

mkdir -p "$OUTPUT_DIR"
stage="$(mktemp -d "$OUTPUT_DIR/.release-stage-XXXXXX")"
cleanup() { rm -rf "$stage"; }
trap cleanup EXIT INT TERM

root="$stage/$RELEASE_NAME"
mkdir -p "$root/assets/lang/zh-Hant" "$root/assets/manual/zh-Hant"
mkdir -p "$root/artwork/modern-icon/m1/trial"
mkdir -p "$root/packaging"

CGO_ENABLED="$TARGET_CGO" GOOS="$TARGET_GOOS" GOARCH="$TARGET_GOARCH" \
  go build -trimpath -ldflags="-s -w" -o "$root/$binary" ./cmd/demonwinter
if [[ ! -s "$root/$binary" ]]; then
  echo "拒絕打包：$TARGET_GOOS/$TARGET_GOARCH 引擎執行檔不存在或為空" >&2
  exit 1
fi

cp README.md "$root/README.md"
cp packaging/README-zh-Hant.txt "$root/開始遊戲.txt"
cp packaging/RELEASE-NOTES-zh-Hant.md "$root/packaging/"
cp assets/lang/zh-Hant/* "$root/assets/lang/zh-Hant/"
cp assets/manual/zh-Hant/* "$root/assets/manual/zh-Hant/"
cp artwork/modern-icon/m1/trial/* "$root/artwork/modern-icon/m1/trial/"

# 授權邊界是所有平台共用的發行契約。
if find "$root" -type f \
  \( -iname "*.DAT" -o -iname "*.DTT" -o -iname "*.SHE" -o -iname "*.SHP" \
     -o -iname "*.PIE" -o -iname "*.PIC" -o -iname "STDFONT.15" \
     -o -iname "SPCFONT.15" \) | grep -q .; then
  echo "拒絕打包：staging 內含原版資料或倚天字型" >&2
  exit 1
fi

package="$OUTPUT_DIR/$RELEASE_NAME.tar.gz"
tar -C "$stage" -czf "$package" "$RELEASE_NAME"
checksum="$package.sha256"
(
  cd "$OUTPUT_DIR"
  filename="$(basename "$package")"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$filename"
  else
    shasum -a 256 "$filename"
  fi
) >"$checksum"

printf "發行包：%s\n" "$package"
printf "校驗碼：%s\n" "$checksum"
