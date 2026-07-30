FROM debian:bookworm-slim

RUN apt-get update -qq \
    && apt-get install -y --no-install-recommends \
       ffmpeg fluid-soundfont-gm fluidsynth fonts-noto-cjk imagemagick \
    && rm -rf /var/lib/apt/lists/*

