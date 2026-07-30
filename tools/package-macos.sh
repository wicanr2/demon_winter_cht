#!/usr/bin/env bash
# 僅能在原生 macOS runner 執行。
set -euo pipefail

: "${VERSION:?缺少 VERSION}"
: "${TARGET_GOARCH:?缺少 TARGET_GOARCH}"
OUTPUT_DIR="${OUTPUT_DIR:-$PWD/dist}"
name="DemonWinter-zh-Hant-${VERSION}-macOS-${TARGET_GOARCH}"
stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT INT TERM
app="$stage/DemonWinter.app"
res="$app/Contents/Resources"
mkdir -p "$app/Contents/MacOS" "$res/assets/lang/zh-Hant" \
  "$res/assets/manual/zh-Hant" "$res/artwork/modern-icon/m1/trial" \
  "$res/docs/audio" "$res/packaging"

CGO_ENABLED=1 GOOS=darwin GOARCH="$TARGET_GOARCH" go build -trimpath -ldflags="-s -w" \
  -o "$app/Contents/MacOS/demonwinter-bin" ./cmd/demonwinter
expected_arch="$TARGET_GOARCH"
if [[ "$TARGET_GOARCH" == "amd64" ]]; then
  expected_arch="x86_64"
fi
binary_archs="$(lipo -archs "$app/Contents/MacOS/demonwinter-bin")"
if [[ "$binary_archs" != "$expected_arch" ]]; then
  echo "拒絕打包：預期 $expected_arch，實際 Mach-O 架構為 $binary_archs" >&2
  exit 1
fi
deps="$res/macOS相依函式庫清單.txt"
otool -L "$app/Contents/MacOS/demonwinter-bin" |
  tail -n +2 | awk '{print $1}' | sort -u >"$deps"
while IFS= read -r dep; do
  case "$dep" in
    /System/Library/*|/usr/lib/*) ;;
    *)
      echo "拒絕打包：macOS 執行檔依賴未隨系統提供的路徑：$dep" >&2
      exit 1
      ;;
  esac
done <"$deps"
cp assets/lang/zh-Hant/* "$res/assets/lang/zh-Hant/"
cp assets/manual/zh-Hant/* "$res/assets/manual/zh-Hant/"
cp artwork/modern-icon/m1/trial/* "$res/artwork/modern-icon/m1/trial/"
cp docs/audio/remake-*.wav "$res/docs/audio/"
cp README.md "$res/README.md"
cp packaging/README-zh-Hant.txt "$res/開始遊戲.txt"
cp packaging/RELEASE-NOTES-zh-Hant.md "$res/packaging/"
cat >"$app/Contents/MacOS/demonwinter" <<'EOF'
#!/bin/sh
set -eu
res="$(CDPATH= cd -- "$(dirname -- "$0")/../Resources" && pwd)"
exec "$(dirname -- "$0")/demonwinter-bin" \
  -lang "$res/assets/lang/zh-Hant" -manual "$res/assets/manual/zh-Hant/manual.txt" \
  -modern-icon-dir "$res/artwork/modern-icon/m1/trial" "$@"
EOF
chmod +x "$app/Contents/MacOS/demonwinter"
cat >"$app/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>demonwinter</string>
<key>CFBundleIdentifier</key><string>tw.wicanr2.demonwinter</string>
<key>CFBundleName</key><string>Demon's Winter 冬之魔</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>${VERSION}</string>
<key>LSMinimumSystemVersion</key><string>12.0</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
EOF

test -z "$(find "$app" -type f \( -iname '*.DAT' -o -iname '*.DTT' -o -iname '*.SHE' \
  -o -iname '*.SHP' -o -iname '*.PIE' -o -iname '*.PIC' -o -iname 'STDFONT.15' \
  -o -iname 'SPCFONT.15' \) -print -quit)"
plutil -lint "$app/Contents/Info.plist"
codesign --force --deep --sign - "$app"
codesign --verify --deep --strict "$app"
"$app/Contents/MacOS/demonwinter-bin" -list-scenes >/dev/null
mkdir -p "$OUTPUT_DIR"
ditto -c -k --sequesterRsrc --keepParent "$app" "$OUTPUT_DIR/$name.zip"
(cd "$OUTPUT_DIR" && shasum -a 256 "$name.zip" >"$name.zip.sha256")
