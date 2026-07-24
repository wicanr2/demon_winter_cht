# Ghidra headless 反組譯環境（Demon's Winter 逆向工程專案）
#
# 用途：對 workplace/orig/demwin/DEMON.INT（MS-DOS MZ, 16-bit real mode）跑
# analyzeHeadless 批次分析。版本寫死以確保可重現。
#
# 版本鎖定（2026-07-24 建置時的 GitHub Releases 最新穩定版）：
#   - Ghidra 12.1.2（tag: Ghidra_12.1.2_build，發佈 2026-06-05）
#   - Ghidra 12.x 系列要求 JDK 21+，故 base image 選 eclipse-temurin:21-jdk-jammy
#
# 建置：
#   docker build -t demwin-ghidra:12.1.2 -f docker/ghidra/Dockerfile docker/ghidra
#
# 不要在系統上另外安裝 Ghidra / JDK —— 全程 docker（見專案 CLAUDE.md 硬規則）。

FROM eclipse-temurin:21-jdk-jammy

ARG GHIDRA_VERSION=12.1.2
ARG GHIDRA_BUILD_DATE=20260605
ARG GHIDRA_SHA256=b62e81a0390618466c019c60d8c2f796ced2509c4c1aea4a37644a77272cf99d
ARG GHIDRA_URL=https://github.com/NationalSecurityAgency/ghidra/releases/download/Ghidra_${GHIDRA_VERSION}_build/ghidra_${GHIDRA_VERSION}_PUBLIC_${GHIDRA_BUILD_DATE}.zip

RUN apt-get update && apt-get install -y --no-install-recommends \
        wget \
        unzip \
        ca-certificates \
        python3 \
    && rm -rf /var/lib/apt/lists/*

RUN wget -q "${GHIDRA_URL}" -O /tmp/ghidra.zip \
    && echo "${GHIDRA_SHA256}  /tmp/ghidra.zip" | sha256sum -c - \
    && unzip -q /tmp/ghidra.zip -d /opt \
    && mv /opt/ghidra_${GHIDRA_VERSION}_PUBLIC /opt/ghidra \
    && rm -f /tmp/ghidra.zip \
    && chmod +x /opt/ghidra/support/analyzeHeadless

ENV GHIDRA_INSTALL_DIR=/opt/ghidra
ENV PATH="${GHIDRA_INSTALL_DIR}/support:${PATH}"

WORKDIR /work

ENTRYPOINT ["/opt/ghidra/support/analyzeHeadless"]
