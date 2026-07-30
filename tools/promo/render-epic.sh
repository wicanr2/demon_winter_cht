#!/usr/bin/env bash
# 容器內確定性合成第二支 72 秒史詩宣傳片。
set -euo pipefail

W=1280
H=720
FPS=25
ROOT=/src
WORK=/work
CAP="$WORK/capture"
TMP="$WORK/render"
OUT="$WORK/out/demon-winter-epic-trailer.mp4"
SCORE="$WORK/score/demon-winter-trailer-master.wav"
TITLE="$ROOT/docs/design/img/demon-winter-modern-title-v1.png"
ORIGINAL_TITLE="$ROOT/docs/images/01-title.png"
FONT=/usr/share/fonts/opentype/noto/NotoSerifCJK-Bold.ttc

mkdir -p "$TMP" "$WORK/out"

common_vf="scale=${W}:${H}:force_original_aspect_ratio=decrease:flags=lanczos,pad=${W}:${H}:(ow-iw)/2:(oh-ih)/2:black,setsar=1,fps=${FPS},format=yuv420p"

title_segment() {
    local out=$1 sec=$2 final=$3 frames
    frames=$((sec * FPS))
    local text=""
    if [ "$final" = yes ]; then
        text=",drawbox=x=0:y=610:w=1280:h=110:color=black@0.62:t=fill,\
drawtext=fontfile=${FONT}:text='1988 史詩角色扮演經典・繁體中文 remake':fontcolor=white:fontsize=30:x=(w-text_w)/2:y=628,\
drawtext=fontfile=${FONT}:text='github.com/wicanr2/demon_winter_cht':fontcolor=0x9fdcff:fontsize=23:x=(w-text_w)/2:y=674"
    fi
    ffmpeg -y -loglevel error -loop 1 -i "$TITLE" -t "$sec" \
        -vf "scale=1340:754:flags=lanczos,zoompan=z='min(zoom+0.00018,1.028)':x='iw/2-(iw/zoom/2)':y='ih/2-(ih/zoom/2)':d=${frames}:s=${W}x${H}:fps=${FPS},fade=t=in:st=0:d=0.5,fade=t=out:st=$(awk "BEGIN{print $sec-0.35}"):d=0.35${text},format=yuv420p" \
        -an -threads 2 -c:v libx264 -preset veryfast -crf 19 "$out"
}

still_history() {
    ffmpeg -y -loglevel error -loop 1 -i "$ORIGINAL_TITLE" -t 5 \
        -vf "${common_vf},drawbox=x=0:y=0:w=1280:h=112:color=black@0.68:t=fill,\
drawtext=fontfile=${FONT}:text='1988':fontcolor=0x9fdcff:fontsize=58:x=66:y=18,\
drawtext=fontfile=${FONT}:text='有限硬體，承載前作 32 倍的伊姆洛斯':fontcolor=white:fontsize=30:x=250:y=36,\
fade=t=in:st=0:d=0.35,fade=t=out:st=4.65:d=0.35" \
        -an -threads 2 -c:v libx264 -preset veryfast -crf 19 "$TMP/s01.mp4"
}

live() {
    local in=$1 out=$2 start=$3 sec=$4 caption=$5
    ffmpeg -y -loglevel error -ss "$start" -i "$in" -t "$sec" \
        -vf "${common_vf},drawbox=x=0:y=650:w=1280:h=70:color=black@0.68:t=fill,\
drawtext=fontfile=${FONT}:text='${caption}':fontcolor=white:fontsize=29:x=(w-text_w)/2:y=668,\
fade=t=in:st=0:d=0.22,fade=t=out:st=$(awk "BEGIN{print $sec-0.22}"):d=0.22" \
        -an -r "$FPS" -threads 2 -c:v libx264 -preset veryfast -crf 19 "$out"
}

title_segment "$TMP/s00.mp4" 3 no
still_history
live "$CAP/f8/f8.mp4" "$TMP/s02.mp4" 4.9 12 \
    "同一局實按 F8　EGA → CGA → Modern Icon → EGA"
live "$CAP/dungeon/dungeon.mp4" "$TMP/s03.mp4" 0 8 \
    "五人深入地城　密門・陷阱・房間敘事"
live "$CAP/town/town.mp4" "$TMP/s04.mp4" 0 8 \
    "城鎮經濟　議價・裝備・神殿・學院"
live "$CAP/boss/boss.mp4" "$TMP/s05.mp4" 1.8 12 \
    "戰術戰鬥　符文法術與原版怪物 AI"
live "$CAP/sea/sea.mp4" "$TMP/s06.mp4" 1.5 10 \
    "跨越冰海　相對轉舵與海戰仍照原規則"
live "$CAP/boss/boss.mp4" "$TMP/s07.mp4" 6.8 8 \
    "面對澤瑞斯　明知代價，仍選擇前進"
title_segment "$TMP/s08.mp4" 6 yes

LIST="$TMP/epic-list.txt"
: > "$LIST"
for n in 00 01 02 03 04 05 06 07 08; do
    printf "file '%s/s%s.mp4'\n" "$TMP" "$n" >> "$LIST"
done

ffmpeg -y -loglevel error -f concat -safe 0 -i "$LIST" \
    -c copy "$TMP/silent.mp4"

# 已核准的 72 秒新編配樂為主體；PC speaker 點音只在年代、F8、戰鬥及決斷轉折出現。
ffmpeg -y -loglevel error -i "$SCORE" \
    -i "$ROOT/docs/audio/06-g3.wav" -i "$ROOT/docs/audio/02-c3.wav" \
    -i "$ROOT/docs/audio/09-c4.wav" -i "$ROOT/docs/audio/00-death.wav" \
    -filter_complex "\
[0:a]volume=0.90[score];\
[1:a]adelay=3000|3000,volume=0.75[a1];\
[2:a]adelay=8100|8100,volume=0.70[a2];\
[1:a]adelay=10500|10500,volume=0.70[a3];\
[3:a]adelay=13200|13200,volume=0.72[a4];\
[1:a]adelay=20500|20500,volume=0.58[a5];\
[3:a]adelay=36500|36500,volume=0.82[a6];\
[2:a]adelay=57500|57500,volume=0.72[a7];\
[4:a]atrim=0:1.6,adelay=58500|58500,volume=0.32[a8];\
[3:a]adelay=64000|64000,volume=0.95[a9];\
[score][a1][a2][a3][a4][a5][a6][a7][a8][a9]amix=inputs=10:duration=first:normalize=0,\
alimiter=limit=0.89,volume=0.78,atrim=0:72[a]" \
    -map "[a]" -ar 48000 -c:a aac -b:a 192k "$TMP/audio.m4a"

ffmpeg -y -loglevel error -i "$TMP/silent.mp4" -i "$TMP/audio.m4a" \
    -map 0:v -map 1:a -c:v copy -c:a copy -movflags +faststart "$OUT"

VDUR=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$OUT")
ADUR=$(ffprobe -v error -select_streams a:0 -show_entries stream=duration -of csv=p=0 "$OUT")
awk -v d="$VDUR" 'BEGIN { if (d < 71.95 || d > 72.10) exit 1 }'
awk -v v="$VDUR" -v a="$ADUR" 'BEGIN { x=v-a; if (x<0) x=-x; if (x > 0.15) exit 1 }'

ffmpeg -hide_banner -i "$OUT" -af ebur128=peak=true -f null - 2> "$TMP/loudness.txt"
grep -A14 'Summary:' "$TMP/loudness.txt"
printf '第二支宣傳片 -> %s（video=%s audio=%s）\n' "$OUT" "$VDUR" "$ADUR"
