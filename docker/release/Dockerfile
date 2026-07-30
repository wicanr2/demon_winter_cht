FROM golang:1.25.12-bookworm

ARG APPIMAGETOOL_SHA256=a6d71e2b6cd66f8e8d16c37ad164658985e0cf5fcaa950c90a482890cb9d13e0
ARG APPIMAGE_RUNTIME_SHA256=1cc49bcf1e2ccd593c379adb17c9f85a36d619088296504de95b1d06215aebbf
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl file imagemagick patchelf zip unzip binutils \
    libasound2-dev libgl1-mesa-dev libx11-dev libxcursor-dev libxi-dev \
    libxinerama-dev libxrandr-dev libxxf86vm-dev xvfb \
 && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL -o /tmp/appimagetool.AppImage \
      https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage \
 && echo "${APPIMAGETOOL_SHA256}  /tmp/appimagetool.AppImage" | sha256sum -c - \
 && chmod +x /tmp/appimagetool.AppImage \
 && cd /opt && /tmp/appimagetool.AppImage --appimage-extract >/dev/null \
 && mv /opt/squashfs-root /opt/appimagetool \
 && ln -s /opt/appimagetool/AppRun /usr/local/bin/appimagetool \
 && rm /tmp/appimagetool.AppImage
RUN curl -fsSL -o /opt/runtime-x86_64 \
      https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-x86_64 \
 && echo "${APPIMAGE_RUNTIME_SHA256}  /opt/runtime-x86_64" | sha256sum -c -
