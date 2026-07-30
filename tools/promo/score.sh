#!/usr/bin/env bash
# 可重現地產生《冬之魔》72 秒宣傳片 MIDI 與 24-bit WAV 母帶。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="$ROOT/workplace/promo/score"
VIDEO_IMAGE=demonwinter-video
mkdir -p "$OUT"

if ! docker image inspect "$VIDEO_IMAGE" >/dev/null 2>&1; then
    docker build --memory 1g -t "$VIDEO_IMAGE" "$ROOT/docker/video"
fi

docker run --rm --network none --memory 768m --cpus 1 --pids-limit 256 \
    -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -e GOCACHE=/tmp/gocache -e GOMODCACHE=/tmp/gomod \
    -v "$ROOT:/src" -v dw-gomod:/tmp/gomod:ro -w /src \
    demonwinter-go go run ./cmd/dwscore \
    -out workplace/promo/score/demon-winter-trailer.mid

docker run --rm --network none --memory 768m --cpus 2 --pids-limit 256 \
    -u "$(id -u):$(id -g)" -v "$OUT:/score" -w /score \
    "$VIDEO_IMAGE" fluidsynth -ni -r 48000 \
    -F /score/demon-winter-trailer-raw.wav \
    /usr/share/sounds/sf2/FluidR3_GM.sf2 \
    /score/demon-winter-trailer.mid

docker run --rm --network none --memory 768m --cpus 2 --pids-limit 256 \
    -u "$(id -u):$(id -g)" -v "$OUT:/score" -w /score \
    "$VIDEO_IMAGE" ffmpeg -y -loglevel error \
    -i demon-winter-trailer-raw.wav \
    -af 'apad,atrim=0:72,aecho=0.8:0.72:83|167:0.16|0.08,loudnorm=I=-16:TP=-1.5:LRA=11,afade=t=in:st=0:d=1.5,afade=t=out:st=69:d=3,atrim=0:72' \
    -ar 48000 -c:a pcm_s24le demon-winter-trailer-master.wav

docker run --rm --network none --memory 512m --cpus 1 --pids-limit 128 \
    -u "$(id -u):$(id -g)" -v "$OUT:/score" -w /score \
    "$VIDEO_IMAGE" bash -lc '
        set -e
        duration=$(ffprobe -v error -show_entries format=duration -of csv=p=0 \
            demon-winter-trailer-master.wav)
        awk -v d="$duration" "BEGIN { if (d < 71.99 || d > 72.01) exit 1 }"
        ffmpeg -hide_banner -i demon-winter-trailer-master.wav \
            -af ebur128=peak=true -f null - 2> loudness.txt
        grep -A14 "Summary:" loudness.txt
        ffmpeg -y -loglevel error -i demon-winter-trailer-master.wav \
            -filter_complex "showwavespic=s=1600x420:colors=0x66ccff" \
            -frames:v 1 waveform.png
        printf "duration=%s\n" "$duration"
    '

printf '配樂母帶 -> %s\n' "$OUT/demon-winter-trailer-master.wav"
