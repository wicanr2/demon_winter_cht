# 117 — 音效完整交叉引用與吐息勘誤

日期：2026-07-30  
輸入：`DEMON.INT`，SHA-256
`fc1df05513bfa0f1a38f95ce0fbe5e6ec390c8192b2f99b3b6118b3c23868ea5`  
工具：IDA Pro 9.4 headless、既有 `workplace/ida/DEMON.INT.i64`、
[`tools/ida_audio_xrefs.idc`](../../tools/ida_audio_xrefs.idc)

## 問題

獨立音訊稽核發現三個疑點：

1. remake 的怪物吐息只要有命中就播放死亡旋律；
2. effect 3／6／7 已合成但沒有 remake 呼叫端；
3. 非戰鬥全滅是否漏播死亡旋律。

## IDA 完整清冊

IDA 對音效選擇器 `sub_20485`（Ghidra `FUN_1d9f_2a95`）列出 **14 個**
直接程式交叉引用，沒有其他 caller：

| 呼叫參數 | 數量 | 位址／用途 |
|---|---:|---|
| 8 | 4 | `14A36`、`14DCD`、`14E6E`、`15465`；特殊傷害、吐息扣血 |
| -1 | 1 | `14F35`；共用戰鬥單位死亡處理 |
| 2 | 6 | `15C7A`、`173B4`、`17A69`、`18B8F`、`19527`、`196C7`；戰鬥 AP 不足 |
| 動態 1／4 | 1 | `16152`；怪物／玩家未命中 |
| 動態 5／8 | 1 | `16380`；武器類型命中 |
| 1 | 1 | `203FD`；聲音開啟時的通用 wrapper |

可重播命令：

```bash
docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  -v "$PWD:/work:ro" -w /tmp ida-pro-9.4-ver2 bash -c \
  'cp /work/workplace/ida/DEMON.INT.i64 /tmp/DEMON.INT.i64;
   idat -A -L/tmp/ida.log -S/work/tools/ida_audio_xrefs.idc /tmp/DEMON.INT.i64;
   grep AUDIO_XREF /tmp/ida.log'
```

## 裁決

### effect 3／6／7

它們是十項音效庫裡可播放但**原版遊戲沒有呼叫的未使用音階**。完整直接
XREF 清冊與兩個動態參數分支已涵蓋所有實際值；remake 不應為填滿音階而虛構
觸發情境。

### 非戰鬥全滅

原版全隊死亡動作 `25be:000c` 不在 14 個 caller 內，也沒有經共用 wrapper
播放 -1。死亡旋律的唯一直接 caller 是戰鬥單位死亡處理 `14F35`。因此
`checkPartyDeath()` 不播放死亡旋律不是漏接，而是原版行為；不修改。

### 怪物吐息

IDA `sub_15088+3D9`（`15465`）證明第二趟命中格先播放 effect 8，再依 HP
決定是否進入共用死亡處理；只有死亡處理才播放 -1。remake 原本把「命中數
非零」直接映成死亡旋律，確實錯誤。

現已改成：

- 無人死亡：effect 8（C4）；
- 至少一人死亡：以死亡旋律作該輪最後可聽結果；
- 不同 effect 播放前會停止其他 player，維持原版單聲道，不疊成和弦。

`breathEffect` 的受傷、免疫、混合死亡三組 helper 測試固定效果選擇；
可注入 `soundOutput` spy 則直接固定 `playSound` 的轉送。這兩層目前沒有
冒充完整 UI 呼叫鏈或真實音效裝置測試；攻擊、吐息、密語、寶珠等端到端
觸發與實機播放仍屬動態驗證邊界。

## 獨立重驗邊界

同日修正後再由另一個獨立 subagent 唯讀重驗目前版本：

- 重跑 IDA 9.4 得到相同 14 個直接 XREF；
- `dwsound` 重建的九份 WAV 與文件檔逐位元組相同；
- 九份皆為 44.1 kHz、16-bit、單聲道 PCM，非靜音，峰值約 −6.02 dB；
- 單音為 0.164762 秒，死亡旋律為 1.537823 秒，休止 tick 符合序列；
- 攻擊四分支、吐息三分支、撞岸與切換相關抽樣測試通過。

因此可稱「原版音訊範圍與技術實作完成」，不可稱「玩家聽感或三平台音效
裝置驗收完成」。後者仍需人耳檢查音高、音色、預設音量、連續事件的單聲道
中斷感、Sound on/off，以及 Linux／macOS／Windows 實機是否正常出聲。
