#!/usr/bin/env bash
set -euo pipefail

: "${VERSION:?缺少 VERSION}"
OUTPUT_DIR="${OUTPUT_DIR:-$PWD/dist}"
name="DemonWinter-zh-Hant-${VERSION}-x86_64"
stage="$(mktemp -d /tmp/dw-appimage-XXXXXX)"
trap 'rm -rf "$stage"' EXIT INT TERM
appdir="$stage/DemonWinter.AppDir"
payload="$appdir/usr/share/demonwinter"
mkdir -p "$appdir/usr/bin" "$appdir/usr/lib" "$payload/assets/lang/zh-Hant" \
  "$payload/assets/manual/zh-Hant" "$payload/artwork/modern-icon/m1/trial" \
  "$payload/docs/audio"

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" \
  -o "$appdir/usr/bin/demonwinter" ./cmd/demonwinter
cp assets/lang/zh-Hant/* "$payload/assets/lang/zh-Hant/"
cp assets/manual/zh-Hant/* "$payload/assets/manual/zh-Hant/"
cp artwork/modern-icon/m1/trial/* "$payload/artwork/modern-icon/m1/trial/"
cp docs/audio/remake-*.wav "$payload/docs/audio/"
cp README.md "$payload/README.md"
cp packaging/README-zh-Hant.txt "$payload/開始遊戲.txt"
cp packaging/demonwinter.svg "$appdir/demonwinter.svg"
ln -s demonwinter.svg "$appdir/.DirIcon"

# 只附非 glibc 的直接動態相依；AppRun 以私有 lib 優先。
ldd "$appdir/usr/bin/demonwinter" |
  awk '{for (i=1;i<=NF;i++) if ($i ~ /^\//) print $i}' |
  while read -r lib; do
    case "$(basename "$lib")" in
      libc.so.*|libm.so.*|libpthread.so.*|libdl.so.*|librt.so.*|ld-linux*) continue ;;
    esac
    cp -L "$lib" "$appdir/usr/lib/"
  done

cat >"$appdir/AppRun" <<'EOF'
#!/bin/sh
set -eu
root="${APPDIR:-$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)}"
payload="$root/usr/share/demonwinter"
export LD_LIBRARY_PATH="$root/usr/lib${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
exec "$root/usr/bin/demonwinter" \
  -lang "$payload/assets/lang/zh-Hant" \
  -manual "$payload/assets/manual/zh-Hant/manual.txt" \
  -modern-icon-dir "$payload/artwork/modern-icon/m1/trial" "$@"
EOF
chmod +x "$appdir/AppRun"
cat >"$appdir/demonwinter.desktop" <<'EOF'
[Desktop Entry]
Type=Application
Name=Demon's Winter 冬之魔
Exec=demonwinter
Icon=demonwinter
Categories=Game;RolePlaying;
Terminal=false
EOF

if find "$appdir" -type f \( -iname '*.DAT' -o -iname '*.DTT' -o -iname '*.SHE' \
  -o -iname '*.SHP' -o -iname '*.PIE' -o -iname '*.PIC' -o -iname 'STDFONT.15' \
  -o -iname 'SPCFONT.15' \) -print -quit | grep -q .; then
  echo "拒絕打包：AppDir 含原版資料或倚天字型" >&2
  exit 1
fi

mkdir -p "$OUTPUT_DIR"
ARCH=x86_64 appimagetool --runtime-file /opt/runtime-x86_64 \
  "$appdir" "$OUTPUT_DIR/$name.AppImage"
chmod +x "$OUTPUT_DIR/$name.AppImage"
(cd "$OUTPUT_DIR" && sha256sum "$name.AppImage" >"$name.AppImage.sha256")
printf 'AppImage：%s\n' "$OUTPUT_DIR/$name.AppImage"
