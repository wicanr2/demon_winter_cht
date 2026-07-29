# 15 — Modern EGA PNG manifest loader 端到端驗證

日期：2026-07-29。

## 1. 目的

單元測試可以證明 loader 會拒絕錯尺寸、錯格數與半透明像素，但不能證明五張
atlas 真能走到 Ebiten 畫面。本輪用現有「完整調色預覽」當內容相同的控制組：

1. 從自備 EGA 素材匯出 102/102/44/240/32 格 PNG theme；
2. 用 `-modern-theme-dir` 重新載入；
3. 與原本記憶體調色路徑在相同座標、seed、按鍵下截圖；
4. 比較完整 PNG bytes。

匯出工具是 `tools/modernpreview`。輸出由原版素材衍生，只能放 `/tmp` 或
`workplace/dump`，不得提交或打包；工具本身不含任何原版資料。

## 2. 可重跑步驟

```bash
tools/go.sh run ./tools/modernpreview \
  -data workplace/orig/demwin/DEM_DATA \
  -out workplace/dump/modern-preview-theme

tools/screenshot.sh /tmp/modern-manifest-smoke.png KEYS=Return \
  -video=modern \
  -modern-theme-dir=workplace/dump/modern-preview-theme \
  -map=34 -x=28 -y=50 -seed=11

tools/screenshot.sh /tmp/modern-memory-smoke.png KEYS=Return \
  -video=modern -map=34 -x=28 -y=50 -seed=11

sha256sum /tmp/modern-manifest-smoke.png /tmp/modern-memory-smoke.png
cmp -s /tmp/modern-manifest-smoke.png /tmp/modern-memory-smoke.png
```

## 3. 結果

兩張 640×400 截圖的 SHA-256 都是：

```text
40f34df05c31719195fd7887408df7c4d2016b67554078b71a157ca644427020
```

`cmp` 回傳 0，證明兩條路徑逐 byte 相同。遊戲 log 另明確記錄：

```text
Modern EGA：使用逐格 PNG theme workplace/dump/modern-preview-theme
```

因此已證明：

- manifest 的五張 sheet 欄位都接到正確 consumer；
- frame 的逐列切片順序、尺寸與索引沒有漂移；
- PNG 路徑不改 640×400 版面、倚天字型、地圖座標與 theme UI；
- 沒有候選目錄時仍走原本的完整調色預覽。

這只驗證**素材管線**，不代表 B 方向逐格美術已核准或完成。

## 4. 清理

驗收後刪除 `workplace/dump/modern-preview-theme` 與兩張臨時索引 atlas。
一擊 Docker 使用 `--rm`；未留下專案容器。只保留可重跑工具與本文件。
