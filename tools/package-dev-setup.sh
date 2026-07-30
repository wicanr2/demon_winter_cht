#!/usr/bin/env bash
# 建立只留在本機 dist-all/ 的私人開發接續包。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="${DEV_SETUP_VERSION:-20260730}"
case "$VERSION" in
    *[!0-9]*|'') echo "DEV_SETUP_VERSION 只能是數字日期，例如 20260730" >&2; exit 1 ;;
esac

NAME="demon-winter-dev-setup-$VERSION"
ARCHIVE="$ROOT/dist-all/$NAME.tar.gz"
IMAGE=demonwinter-release

git check-ignore -q dist-all/.dev-setup-sentinel || {
    echo "拒絕建立：dist-all 未被 Git 忽略" >&2
    exit 1
}
git diff --check
test -z "$(git status --porcelain --untracked-files=all)" || {
    echo "拒絕建立：工作樹有尚未提交的變更；Git bundle 只會收已提交內容" >&2
    git status --short >&2
    exit 1
}

docker image inspect "$IMAGE" >/dev/null 2>&1 || {
    echo "找不到 $IMAGE；先依 docker/release/Dockerfile 建置" >&2
    exit 1
}

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
    -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -e NAME="$NAME" \
    -v "$ROOT:/repo:ro" \
    -v "$ROOT/.git:/repo/.git:ro" \
    -v "$ROOT/dist-all:/output" \
    -w /repo "$IMAGE" bash -c '
set -euo pipefail
rm -f "/output/$NAME.tar.gz" "/output/$NAME.tar.gz.sha256"
for required in \
  /repo/workplace/orig/demwin/DEMON.EXE \
  /repo/workplace/orig/demwin/DEMON.INT \
  /repo/workplace/orig/demwin/DEM_DATA/FILES.DAT \
  /repo/workplace/orig/demwin/DEM_DATA/SUM.MAP \
  /repo/workplace/eten/STDFONT.15 \
  /repo/workplace/eten/SPCFONT.15; do
  test -s "$required" || { echo "缺少私人 dev-setup 輸入：$required" >&2; exit 1; }
done
stage=$(mktemp -d /tmp/dw-dev-setup-XXXXXX)
trap "rm -rf \"$stage\"" EXIT INT TERM
root="$stage/$NAME"
mkdir -p "$root/private/original" "$root/private/fonts/etan_font"

git config --global --add safe.directory /repo
git bundle create "$root/demon-winter-repo.bundle" --all
git bundle verify "$root/demon-winter-repo.bundle"
git rev-parse HEAD >"$root/HEAD.txt"
git status --short >"$root/GIT-STATUS.txt"

cp -a /repo/workplace/orig/demwin "$root/private/original/"
cp -a /repo/workplace/eten/. "$root/private/fonts/etan_font/"

cat >"$root/bootstrap.sh" <<"EOF"
#!/usr/bin/env bash
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
DEST="${1:-$PWD/demon_winter}"
test ! -e "$DEST" || { echo "拒絕覆寫既有目的地：$DEST" >&2; exit 1; }
git clone "$HERE/demon-winter-repo.bundle" "$DEST"
mkdir -p "$DEST/workplace/orig" "$DEST/workplace/eten"
cp -a "$HERE/private/original/demwin" "$DEST/workplace/orig/"
cp -a "$HERE/private/fonts/etan_font/." "$DEST/workplace/eten/"
printf "已還原到 %s\n下一步：cd %q && bash tools/dev-setup.sh\n" "$DEST" "$DEST"
EOF
chmod +x "$root/bootstrap.sh"

cat >"$root/REBUILD.md" <<EOF
# Demon Winter 私人 dev-setup

此包包含完整 Git bundle、合法原版 DOS oracle／DEM_DATA 與私人倚天字型。
含版權內容，只供本機接續開發，不得上傳 GitHub、CI artifact 或公開雲端。

## 還原

\`\`\`bash
tar -xzf $NAME.tar.gz
cd $NAME
bash bootstrap.sh /path/to/demon_winter
cd /path/to/demon_winter
bash tools/dev-setup.sh
\`\`\`

封包來源提交：\`$(git rev-parse HEAD)\`
公開開發指南還原後位於：\`docs/DEV_SETUP.md\`
EOF

(
  cd "$root"
  find . -type f ! -name MANIFEST.sha256 -print0 |
    sort -z | xargs -0 sha256sum >MANIFEST.sha256
)

test "$(find "$root/private/original/demwin/DEM_DATA" -maxdepth 1 -type f | wc -l)" -eq 94
test -s "$root/private/original/demwin/DEMON.INT"
test -s "$root/private/fonts/etan_font/STDFONT.15"

tar -C "$stage" -czf "/output/$NAME.tar.gz" "$NAME"
cd /output
sha256sum "$NAME.tar.gz" >"$NAME.tar.gz.sha256"
'

printf '私人 dev-setup 已建立：%s\n' "$ARCHIVE"
