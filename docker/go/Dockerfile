# Go + Ebiten 建置／測試環境
#
# 硬規則：本專案一律用 docker 編譯，不污染主機 Go 環境。
# Ebiten 需要 X11 / OpenGL 的標頭檔才能編譯（即使跑 headless 測試也要，
# 因為 cmd/demonwinter 會 import ebiten）。
FROM golang:1.22-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
        libgl1-mesa-dev \
        libxcursor-dev \
        libxi-dev \
        libxinerama-dev \
        libxrandr-dev \
        libxxf86vm-dev \
        libasound2-dev \
        pkg-config \
    && rm -rf /var/lib/apt/lists/*

# module cache 掛在 volume 上，避免每次 rebuild 重抓
ENV GOPATH=/go
ENV GOFLAGS=-buildvcs=false
ENV CGO_ENABLED=1

WORKDIR /src
