#!/usr/bin/env bash
set -euo pipefail

: "${VERSION:?缺少 VERSION}"
OUTPUT_DIR="${OUTPUT_DIR:-$PWD/dist}"
name="DemonWinter-zh-Hant-${VERSION}-windows-x86_64"
stage="$(mktemp -d /tmp/dw-windows-XXXXXX)"
trap 'rm -rf "$stage"' EXIT INT TERM
root="$stage/$name"
mkdir -p "$root/assets/lang/zh-Hant" "$root/assets/manual/zh-Hant" \
  "$root/artwork/modern-icon/m1/trial"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" \
  -o "$root/demonwinter.exe" ./cmd/demonwinter
cp assets/lang/zh-Hant/* "$root/assets/lang/zh-Hant/"
cp assets/manual/zh-Hant/* "$root/assets/manual/zh-Hant/"
cp artwork/modern-icon/m1/trial/* "$root/artwork/modern-icon/m1/trial/"
cp README.md "$root/README.md"
cp packaging/README-zh-Hant.txt "$root/開始遊戲.txt"

imports="$stage/imports.txt"
objdump -p "$root/demonwinter.exe" | awk '/DLL Name:/ {print tolower($3)}' | sort -u >"$imports"
allowed='^(advapi32|bcrypt|comdlg32|dwmapi|gdi32|imm32|kernel32|ntdll|ole32|oleaut32|opengl32|shell32|shlwapi|user32|version|winmm|ws2_32)\.dll$'
if grep -Ev "$allowed" "$imports" | grep -q .; then
  echo "拒絕打包：發現未隨包附上的第三方 DLL import：" >&2
  grep -Ev "$allowed" "$imports" >&2
  exit 1
fi
cp "$imports" "$root/Windows系統DLL清單.txt"
printf '第三方 DLL：0（所有 import 均為 Windows 系統 DLL）\n' >"$root/第三方DLL說明.txt"

if find "$root" -type f \( -iname '*.DAT' -o -iname '*.DTT' -o -iname '*.SHE' \
  -o -iname '*.SHP' -o -iname '*.PIE' -o -iname '*.PIC' -o -iname 'STDFONT.15' \
  -o -iname 'SPCFONT.15' \) -print -quit | grep -q .; then
  echo "拒絕打包：Windows staging 含原版資料或倚天字型" >&2
  exit 1
fi

mkdir -p "$OUTPUT_DIR"
(cd "$stage" && zip -qr "$OUTPUT_DIR/$name.zip" "$name")
(cd "$OUTPUT_DIR" && sha256sum "$name.zip" >"$name.zip.sha256")
printf 'Windows ZIP：%s\n' "$OUTPUT_DIR/$name.zip"
