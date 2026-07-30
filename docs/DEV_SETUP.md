# 開發接續指南（dev-setup）

這份文件讓未來的維護者從零接手《Demon's Winter／冬之魔》繁體中文 remake。
本專案不是 ScummVM patch-only 專案；Go／Ebiten 引擎、翻譯、Modern Icon、
逆向文件、Docker recipes 與驗收工具都在同一個 Git 儲存庫。原版遊戲及倚天
字型仍屬私人輸入，不進 GitHub。

## 只需要什麼

- Docker 與 Git。
- 一份合法的 DOS 版《Demon's Winter》：
  - `DEMON.EXE`
  - `DEMON.INT`
  - 完整 `DEM_DATA/`
- 自備倚天 16×15 字型：
  - `STDFONT.15`
  - `SPCFONT.15`
  - `SPCFSUPP.15` 可選，但私人接續包會一併保存。

所有編譯、測試、截圖、錄影與逆向工具都在 Docker 內執行，不需要在主機安裝 Go、
Xvfb、FFmpeg、Ghidra 或 DOSBox。

## 目錄放置

```text
workplace/
├── orig/demwin/
│   ├── DEMON.EXE
│   ├── DEMON.INT
│   ├── dosbox.conf
│   └── DEM_DATA/
│       ├── FILES.DAT
│       ├── SUM.MAP
│       ├── MONSTER.DAT
│       └── …完整 94 個檔案
└── eten/
    ├── STDFONT.15
    ├── SPCFONT.15
    └── SPCFSUPP.15
```

`workplace/` 已由 Git 忽略。不要把原版檔案或字型複製到 `assets/`、`artwork/`
或任何受版本控制路徑。

## 一鍵驗證開發環境

```bash
bash tools/dev-setup.sh
```

它會：

1. 檢查 Docker、原版 oracle、完整資料與字型。
2. 建置目前釘定的 `demonwinter-go` 工具鏈映像。
3. 在 Xvfb 容器內執行 `go test ./...`。
4. 執行 `dwstrings check`（500/500）與 `dwstrings uicheck`。
5. 執行 `git diff --check`，並確認沒有專案容器殘留。

已有目前工具鏈映像，只想重跑驗證：

```bash
bash tools/dev-setup.sh --rebuild-only
```

## 私人 dev-setup 接續包

```bash
DEV_SETUP_VERSION=20260730 bash tools/package-dev-setup.sh
```

產物只會放在被 Git 忽略的 `dist-all/`：

```text
dist-all/demon-winter-dev-setup-20260730.tar.gz
dist-all/demon-winter-dev-setup-20260730.tar.gz.sha256
```

私人包包含：

- `demon-winter-repo.bundle`：完整 Git 歷史、分支及目前提交；
- `private/original/demwin/`：原版執行檔、設定及完整 `DEM_DATA`；
- `private/fonts/etan_font/`：倚天 16×15 字型；
- `bootstrap.sh`：從 bundle 還原工作樹並把私人輸入放回正確位置；
- `REBUILD.md`、HEAD、Git 狀態與逐檔 SHA-256 manifest。

它不包含 Docker image／volume、建置快取、`dist/`、其他 release、測試輸出、
宣傳片未剪輯母帶、IDA 安裝目錄或原始 `.git/` 目錄。Docker recipes 與 IDA
使用說明已在 Git bundle 中；IDA 9.4 本體仍由開發主機另外提供。

這個私人包含版權資料，**不得上傳 GitHub Release、CI artifact 或公開雲端**。

## 從私人包還原

```bash
tar -xzf demon-winter-dev-setup-20260730.tar.gz
cd demon-winter-dev-setup-20260730
bash bootstrap.sh /path/to/new/demon_winter
cd /path/to/new/demon_winter
bash tools/dev-setup.sh
```

`bootstrap.sh` 拒絕覆寫既有目的地。還原後的原版資料與字型仍位於 Git 忽略的
`workplace/`。

## 常用入口

| 目的 | 指令 |
|---|---|
| Go 測試 | `docker run --rm … demonwinter-go go test ./...`，完整參數見 `tools/dev-setup.sh` |
| 單張畫面 | `tools/screenshot.sh /tmp/shot.png -scene armory -video modern` |
| A6 路徑 | `tools/playthrough.sh tools/playthrough/a6-leg1.txt /tmp/a6-leg1 …` |
| 三主題 P4 | `tools/p4-capture.sh /tmp/p4` |
| 第二支宣傳片 | `bash tools/promo/capture-epic.sh && bash tools/promo/make-epic.sh` |
| 公開乾淨 release | `tools/package-release.sh` |
| 私人四平台完整版 | `tools/package-full-local.sh` |
| IDA／反組譯 | `docs/re/`、`tools/ida_audio_xrefs.idc`、`docker/ghidra/` |

## 接手閱讀順序

1. `AGENTS.md`
2. `CONTEXT.md` §7 最新基線
3. `README.md`
4. `docs/engineering/README.md`
5. 當前工作直接相關的 `docs/re/`、`docs/design/` 或 `docs/playtest/`

不要從舊 checklist 重新開啟已經由位址、實機或測試解決的逆向問題。
