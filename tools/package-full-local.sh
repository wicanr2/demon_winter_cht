#!/usr/bin/env bash
# 從已驗證的公開乾淨包建立只留在本機 dist-all/ 的可直接遊玩完整版。
# 本腳本絕不由 CI 呼叫，也不會修改公開包；原版資料與倚天字型只複製到忽略目錄。
set -euo pipefail

: "${VERSION:?缺少 VERSION，例如 0.1.0}"
: "${PUBLIC_DIST_DIR:?缺少 PUBLIC_DIST_DIR（四平台公開包所在目錄）}"
: "${ORIGINAL_DATA_DIR:?缺少 ORIGINAL_DATA_DIR（合法原版 DEM_DATA）}"
: "${ETEN_FONT_DIR:?缺少 ETEN_FONT_DIR（含 STDFONT.15／SPCFONT.15）}"

OUTPUT_DIR="${OUTPUT_DIR:-$PWD/dist-all}"
repo_root="$(pwd)"
case "$OUTPUT_DIR" in
  "$repo_root/dist-all"|"$repo_root/dist-all/"*) ;;
  *)
    echo "拒絕建立完整版：OUTPUT_DIR 必須位於 $repo_root/dist-all" >&2
    exit 1
    ;;
esac
mkdir -p "$OUTPUT_DIR"
git check-ignore -q "$repo_root/dist-all/.private-release-sentinel" || {
  echo "拒絕建立完整版：dist-all 未被 Git 忽略" >&2
  exit 1
}

for required in \
  "$ORIGINAL_DATA_DIR/PARTY.DAT" \
  "$ORIGINAL_DATA_DIR/FILES.DAT" \
  "$ORIGINAL_DATA_DIR/FILES.DTT" \
  "$ORIGINAL_DATA_DIR/SUM.MAP" \
  "$ORIGINAL_DATA_DIR/MONSTER.DAT" \
  "$ETEN_FONT_DIR/STDFONT.15" \
  "$ETEN_FONT_DIR/SPCFONT.15"; do
  test -s "$required" || {
    echo "缺少完整版必要檔案：$required" >&2
    exit 1
  }
done

appimage="$PUBLIC_DIST_DIR/DemonWinter-zh-Hant-${VERSION}-x86_64.AppImage"
windows_zip="$PUBLIC_DIST_DIR/DemonWinter-zh-Hant-${VERSION}-windows-x86_64.zip"
mac_amd64_zip="$PUBLIC_DIST_DIR/DemonWinter-zh-Hant-${VERSION}-macOS-amd64.zip"
mac_arm64_zip="$PUBLIC_DIST_DIR/DemonWinter-zh-Hant-${VERSION}-macOS-arm64.zip"
for package in "$appimage" "$windows_zip" "$mac_amd64_zip" "$mac_arm64_zip"; do
  test -s "$package" || {
    echo "缺少公開平台包：$package" >&2
    exit 1
  }
done

work="$(mktemp -d /tmp/dw-full-local-XXXXXX)"
trap 'rm -rf "$work"' EXIT INT TERM

copy_private_data() {
  local root="$1"
  mkdir -p "$root/original/DEM_DATA" "$root/fonts/etan_font"
  cp -a "$ORIGINAL_DATA_DIR/." "$root/original/DEM_DATA/"
  cp -a "$ETEN_FONT_DIR/." "$root/fonts/etan_font/"
}

linux_name="DemonWinter-zh-Hant-${VERSION}-linux-x86_64-full-local"
linux_root="$work/$linux_name"
mkdir -p "$linux_root"
cp "$appimage" "$linux_root/DemonWinter.AppImage"
chmod +x "$linux_root/DemonWinter.AppImage"
copy_private_data "$linux_root"
cat >"$linux_root/開始完整版.sh" <<'EOF'
#!/bin/sh
set -eu
base="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
exec "$base/DemonWinter.AppImage" \
  -data "$base/original/DEM_DATA" -eten "$base/fonts/etan_font" "$@"
EOF
chmod +x "$linux_root/開始完整版.sh"
tar -C "$work" -czf "$OUTPUT_DIR/$linux_name.tar.gz" "$linux_name"

unzip -q "$windows_zip" -d "$work/windows"
windows_public="$work/windows/DemonWinter-zh-Hant-${VERSION}-windows-x86_64"
windows_name="DemonWinter-zh-Hant-${VERSION}-windows-x86_64-full-local"
windows_root="$work/$windows_name"
mv "$windows_public" "$windows_root"
copy_private_data "$windows_root"
cat >"$windows_root/開始完整版.bat" <<'EOF'
@echo off
cd /d "%~dp0"
demonwinter.exe -data "%~dp0original\DEM_DATA" -eten "%~dp0fonts\etan_font" %*
EOF
cp "$windows_root/開始完整版.bat" "$windows_root/start-full.bat"
cat >>"$windows_root/README.md" <<'EOF'

## Windows 完整版啟動

解壓後請執行 `start-full.bat`。套件也保留同內容的 `開始完整版.bat`，但部分
舊式解壓工具、非中文系統或相容層可能無法正確保留或辨識中文檔名，因此
`start-full.bat` 是建議使用的相容入口。兩者都會以套件內的合法原版資料與
倚天字型啟動遊戲。
EOF
(cd "$work" && zip -qr "$OUTPUT_DIR/$windows_name.zip" "$windows_name")

package_mac() {
  local arch="$1"
  local input="$2"
  local name="DemonWinter-zh-Hant-${VERSION}-macOS-${arch}-full-local"
  local root="$work/$name"
  mkdir -p "$root"
  unzip -q "$input" -d "$root"
  copy_private_data "$root"
  cat >"$root/開始完整版.command" <<'EOF'
#!/bin/sh
set -eu
base="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
res="$base/DemonWinter.app/Contents/Resources"
exec "$base/DemonWinter.app/Contents/MacOS/demonwinter-bin" \
  -lang "$res/assets/lang/zh-Hant" \
  -manual "$res/assets/manual/zh-Hant/manual.txt" \
  -modern-icon-dir "$res/artwork/modern-icon/m1/trial" \
  -data "$base/original/DEM_DATA" -eten "$base/fonts/etan_font" "$@"
EOF
  chmod +x "$root/開始完整版.command"
  (cd "$work" && zip -qry "$OUTPUT_DIR/$name.zip" "$name")
}
package_mac amd64 "$mac_amd64_zip"
package_mac arm64 "$mac_arm64_zip"

(
  cd "$OUTPUT_DIR"
  sha256sum \
    "$linux_name.tar.gz" \
    "$windows_name.zip" \
    "DemonWinter-zh-Hant-${VERSION}-macOS-amd64-full-local.zip" \
    "DemonWinter-zh-Hant-${VERSION}-macOS-arm64-full-local.zip" \
    >"SHA256SUMS-${VERSION}-full-local.txt"
)

printf '本地完整版已建立於：%s\n' "$OUTPUT_DIR"
