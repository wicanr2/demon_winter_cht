#!/usr/bin/env bash
# 容器內的確定性影片合成。不要直接執行；由 make.sh 掛載素材後呼叫。
set -euo pipefail

W=1280
H=720
FPS=25
TMP=/work/render
CAP=/work/capture
SFX=/work/sound
OUT=/work/out/demon-winter-cht-promo.mp4
FONT_TITLE=/usr/share/fonts/opentype/noto/NotoSerifCJK-Bold.ttc
FONT_BODY=/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc

# 從遊戲本身的 EGA UI 萃取：純黑底、暗紅重點、白字、棕金邊。
BLACK='#000000'
RED='#aa0000'
GOLD='#aa5500'
WHITE='#f2f2f2'
DIM='#aaaaaa'

mkdir -p "$TMP" /work/out

frame_bg() {
    convert -size "${W}x${H}" "xc:$BLACK" \
        -stroke "$WHITE" -strokewidth 2 -fill none \
        -draw "rectangle 34,30 1245,689 rectangle 42,38 1237,681" \
        -stroke "$RED" -strokewidth 3 \
        -draw "line 70,92 1210,92 line 70,628 1210,628" "$1"
}

card() {
    local out=$1 title=$2 sub=$3 foot=$4
    frame_bg "$TMP/bg.png"
    convert "$TMP/bg.png" \
        -font "$FONT_TITLE" -gravity center \
        -fill "$GOLD" -pointsize 82 -annotate +3-37 "$title" \
        -fill "$WHITE" -pointsize 82 -annotate +0-40 "$title" \
        -font "$FONT_BODY" -fill "$RED" -pointsize 36 -annotate +0+58 "$sub" \
        -fill "$DIM" -pointsize 23 -annotate +0+252 "$foot" "$out"
}

slide() {
    local out=$1 shot=$2 caption=$3
    convert "$shot" -resize "${W}x${H}^" -gravity center -extent "${W}x${H}" \
        -fill '#000000bb' -draw "rectangle 0,620 ${W},720" \
        -stroke "$RED" -strokewidth 3 -fill none -draw "line 0,620 ${W},620" \
        -stroke none -font "$FONT_BODY" -fill "$WHITE" -gravity south \
        -pointsize 34 -annotate +0+30 "$caption" "$out"
}

still_video() {
    local img=$1 out=$2 sec=$3
    local fadeout
    fadeout=$(awk "BEGIN { print $sec - 0.45 }")
    ffmpeg -y -loglevel error -loop 1 -i "$img" -t "$sec" -r "$FPS" \
        -vf "fade=t=in:st=0:d=0.45,fade=t=out:st=$fadeout:d=0.45,format=yuv420p" \
        -threads 2 -c:v libx264 -preset veryfast -pix_fmt yuv420p "$out"
}

live_video() {
    local in=$1 out=$2 start=$3 sec=$4
    local fadeout
    fadeout=$(awk "BEGIN { print $sec - 0.35 }")
    ffmpeg -y -loglevel error -ss "$start" -i "$in" -t "$sec" \
        -vf "scale=${W}:${H}:flags=neighbor,fade=t=in:st=0:d=0.35,fade=t=out:st=$fadeout:d=0.35,format=yuv420p" \
        -an -r "$FPS" -threads 2 -c:v libx264 -preset veryfast -pix_fmt yuv420p "$out"
}

card "$TMP/00.png" '冬之魔' "Demon's Winter" '1988 經典角色扮演遊戲・繁體中文 remake'
card "$TMP/02.png" '不只翻譯文字' '重建玩法，也保留原作性格' '原版資料驅動・可重現驗收'
slide "$TMP/03.png" "$CAP/world/01-party.png" '五人隊伍・種族、職業與成長'
slide "$TMP/04.png" "$CAP/world/02-camp.png" '紮營、分裝、施法與野外技能'
slide "$TMP/05.png" "$CAP/world/03-view-land.png" '以原版資料重現探索與視野'
card "$TMP/07.png" '真正能玩的戰鬥' '走位・命中・傷害・戰利品' '每一場遭遇都能由 trace 重播查證'
card "$TMP/08.png" '原汁原味的聲音' '沒有假配樂，只有 PC speaker' '倚天 16×15 粗體・EGA／CGA 雙模式'
card "$TMP/09.png" '冬之魔・繁體中文版' '翻譯・remake・玩法考證' 'github.com/wicanr2/demon_winter_cht'

still_video "$TMP/00.png" "$TMP/s00.mp4" 6
live_video "$CAP/world/world.mp4" "$TMP/s01.mp4" 3.4 9
still_video "$TMP/02.png" "$TMP/s02.mp4" 4
still_video "$TMP/03.png" "$TMP/s03.mp4" 5
still_video "$TMP/04.png" "$TMP/s04.mp4" 5
still_video "$TMP/05.png" "$TMP/s05.mp4" 5
live_video "$CAP/battle/battle.mp4" "$TMP/s06.mp4" 1.6 10
still_video "$TMP/07.png" "$TMP/s07.mp4" 4
still_video "$TMP/08.png" "$TMP/s08.mp4" 5
still_video "$TMP/09.png" "$TMP/s09.mp4" 8

LIST="$TMP/list.txt"
: > "$LIST"
for n in 00 01 02 03 04 05 06 07 08 09; do
    printf "file '%s/s%s.mp4'\n" "$TMP" "$n" >> "$LIST"
done
ffmpeg -y -loglevel error -f concat -safe 0 -i "$LIST" \
    -threads 2 -c:v libx264 -preset veryfast -pix_fmt yuv420p "$TMP/silent.mp4"

DUR=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$TMP/silent.mp4")
# 原作沒有 BGM。建立等長靜音底，再只在切換與戰鬥段疊入原版方波效果。
ffmpeg -y -loglevel error -f lavfi -i "anullsrc=r=44100:cl=mono" -t "$DUR" \
    -i "$SFX/06-g3.wav" -i "$SFX/02-c3.wav" -i "$SFX/09-c4.wav" \
    -filter_complex \
    "[0:a]atrim=0:$DUR[base];\
[1:a]adelay=6000|6000[a1];[2:a]adelay=15000|15000[a2];\
[3:a]adelay=34000|34000[a3];[1:a]adelay=36500|36500[a4];\
[2:a]adelay=39000|39000[a5];[3:a]adelay=42000|42000[a6];\
[base][a1][a2][a3][a4][a5][a6]amix=inputs=7:duration=first:normalize=0,\
afade=t=in:st=0:d=0.5[a]" \
    -map "[a]" -c:a aac -b:a 128k "$TMP/audio.m4a"

ffmpeg -y -loglevel error -i "$TMP/silent.mp4" -i "$TMP/audio.m4a" \
    -map 0:v -map 1:a -c:v copy -c:a copy -movflags +faststart "$OUT"

VDUR=$(ffprobe -v error -select_streams v:0 -show_entries stream=duration -of csv=p=0 "$OUT")
ADUR=$(ffprobe -v error -select_streams a:0 -show_entries stream=duration -of csv=p=0 "$OUT")
awk -v d="$VDUR" 'BEGIN { if (d < 60 || d > 75) exit 1 }'
awk -v v="$VDUR" -v a="$ADUR" 'BEGIN { x=v-a; if (x<0) x=-x; if (x > 0.15) exit 1 }'
ffmpeg -hide_banner -i "$OUT" -af volumedetect -f null - 2>&1 |
    grep -E 'mean_volume|max_volume'
printf 'duration video=%s audio=%s\n' "$VDUR" "$ADUR"
