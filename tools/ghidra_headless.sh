#!/usr/bin/env bash
# tools/ghidra_headless.sh
#
# 封裝 Ghidra headless 分析（docker）。給後續反組譯 agent 用，一行指令跑完
# 匯入 + 自動分析 + （選用）post-script。
#
# 用法：
#   tools/ghidra_headless.sh <binary相對於workplace/的路徑> <專案名稱> [post-script 檔名] [額外參數...]
#
# 範例（對 DEMON.INT 跑完整分析並用內建匯出腳本匯出）：
#   tools/ghidra_headless.sh orig/demwin/DEMON.INT demwin_int ExportAnalysis.java
#
# 注意：post-script 只支援 Java（.java），不支援 Python（.py）——Ghidra 12.x
# 拿掉了內建 Jython，.py script 需要另外裝 PyGhidra 才能跑，本環境沒配這層，
# 詳見 docs/re/00-ghidra-setup.md「踩到的雷」。
#
# 範例（只跑自動分析，不跑 post-script）：
#   tools/ghidra_headless.sh orig/demwin/DEMON.INT demwin_int
#
# 環境需求：docker、已建置好 image（見 docker/ghidra/Dockerfile）：
#   docker build -t demwin-ghidra:12.1.2 -f docker/ghidra/Dockerfile docker/ghidra
#
# 掛載：
#   workplace/          → /work         （唯讀不強制，binary 一般不會被腳本改動）
#   workplace/ghidra/    → /work/ghidra   （Ghidra 專案檔存放處，.gitignore 已排除）
#   workplace/ghidra/export → /work/export（匯出檔輸出目錄，post-script 讀 DW_EXPORT_DIR）
#   tools/ghidra_scripts/ → /work/scripts （自訂 post-script 掛載點）
#
# 16-bit real mode 專用：DEMON.INT / DEMON.EXE 這類 MS-DOS MZ real-mode 執行檔，
# 強制指定 -processor x86:LE:16:Real Mode（Ghidra 內建 MZ loader 有時會誤判成
# 32-bit protected mode，必須明確指定才會用 segment:offset 定址）。
# 如果之後要分析非 16-bit real mode 的其他 binary，用 GHIDRA_PROCESSOR 環境變數覆蓋。

set -euo pipefail

IMAGE="${GHIDRA_IMAGE:-demwin-ghidra:12.1.2}"
PROCESSOR="${GHIDRA_PROCESSOR:-x86:LE:16:Real Mode}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
WORKPLACE_DIR="${PROJECT_ROOT}/workplace"
GHIDRA_PROJ_DIR="${WORKPLACE_DIR}/ghidra"
EXPORT_DIR="${WORKPLACE_DIR}/ghidra/export"
SCRIPTS_DIR="${PROJECT_ROOT}/tools/ghidra_scripts"

if [[ $# -lt 2 ]]; then
    echo "用法: $0 <binary相對於workplace/的路徑> <專案名稱> [post-script 檔名] [額外 analyzeHeadless 參數...]" >&2
    echo "範例: $0 orig/demwin/DEMON.INT demwin_int ExportAnalysis.py" >&2
    exit 1
fi

BINARY_REL="$1"
PROJECT_NAME="$2"
POST_SCRIPT="${3:-}"
shift 2 || true
if [[ -n "${POST_SCRIPT}" ]]; then
    shift || true
fi
EXTRA_ARGS=("$@")

BINARY_HOST_PATH="${WORKPLACE_DIR}/${BINARY_REL}"
if [[ ! -f "${BINARY_HOST_PATH}" ]]; then
    echo "找不到目標 binary: ${BINARY_HOST_PATH}" >&2
    exit 1
fi

HOME_DIR="${GHIDRA_PROJ_DIR}/.home"
mkdir -p "${GHIDRA_PROJ_DIR}" "${EXPORT_DIR}" "${HOME_DIR}"

BINARY_IN_CONTAINER="/work/${BINARY_REL}"

# --user 用 host 端目前使用者的 uid:gid 跑容器，匯出的檔案才會是使用者自己的，
# 不會變成 root 擁有（docker 預設容器內是 root，寫出來的檔案在 host 上也是 root:root，
# 之後其他 agent/使用者用一般權限讀寫會卡權限）。HOME 另外指到 workplace/ghidra/.home，
# 讓 Ghidra 寫 preferences/log 時有地方可寫。
DOCKER_ARGS=(
    run --rm
    --user "$(id -u):$(id -g)"
    -e "HOME=/work/ghidra/.home"
    -v "${WORKPLACE_DIR}:/work"
    -v "${SCRIPTS_DIR}:/scripts"
    -e "DW_EXPORT_DIR=/work/ghidra/export"
    --entrypoint /opt/ghidra/support/analyzeHeadless
    "${IMAGE}"
    /work/ghidra "${PROJECT_NAME}"
    -import "${BINARY_IN_CONTAINER}"
    -processor "${PROCESSOR}"
    -overwrite
)

if [[ -n "${POST_SCRIPT}" ]]; then
    DOCKER_ARGS+=(-postScript "${POST_SCRIPT}" -scriptPath /scripts)
fi

DOCKER_ARGS+=("${EXTRA_ARGS[@]}")

echo "[ghidra_headless] docker ${DOCKER_ARGS[*]}"
docker "${DOCKER_ARGS[@]}"
