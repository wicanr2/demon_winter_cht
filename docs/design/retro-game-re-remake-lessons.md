# 老遊戲反組譯與 remake：可複用模板與《冬之魔》經驗

> 日期：2026-07-29  
> 對應 Codex skill：`reverse-engineer-retro-game-remake`

## 核心結論

老遊戲 remake 不是「把反編譯 C 改到能編譯」，而是建立三條可以互相校正的鏈：

```text
原版 binary／資料 ── 位址與位元組證據 ── 原版實跑 oracle
          │                       │
          └──── 乾淨 parser／規則／UI／存檔 ──── remake 實跑
```

反組譯負責回答原版怎麼做；資料差分與原版實跑負責裁決靜態分析的不確定性；
remake 則用型別、測試和清楚的模組重新表達行為。三者缺一都容易得到「內部測試全綠，
但其實不是原版行為」的結果。

## 建議專案模板

每個新專案一開始就建立四份活文件：

| 文件 | 用途 |
|---|---|
| `RESEARCH-LOG.md` | 一次只回答一個逆向問題，記 input hash、工具版本、位址、證據等級與重現命令 |
| `REMAKE-PLAN.md` | 原版資料邊界、架構、垂直切片、worklist 與刻意差異 |
| `VERIFICATION-MATRIX.md` | 每個子系統的單元測試、原版 oracle、視覺 oracle、正常玩家路徑與發布閘門 |
| `HANDOFF.md` | HEAD、worktree、已驗證事實、剩餘類型、下一個最小動作與不要重踩的路 |

正式模板已收進共用 skill 的 `assets/project-template/`。

## 七階段流程

1. **盤點與封存**：hash executable／data／save／各平台版本；原版資料唯讀，實驗用副本。
2. **建立位址模型**：解析 executable header、relocation、load base；用至少三個字串回讀驗證。
3. **資料格式垂直切片**：先解一筆、dump 一張圖、實跑一條 consumer，再批次處理。
4. **乾淨重寫**：UI → game adapter/rules/data → runtime/render/grid/storage/RNG，依賴單向。
5. **原版對拍**：固定 save、座標、時間、seed、theme；靜態推不動就做相鄰邊界值實驗。
6. **正常玩家路徑**：不用瞬移、發道具、強制勝利；驗建角、移動、戰鬥、存讀與可達性。
7. **打包與交接**：原版資料不散布；跑完整品質閘；清除與新證據衝突的舊 worklist。

## 《冬之魔》最值得複用的教訓

### 數學整除不代表圖形格式正確

精靈尺寸曾出現 32×16、16×32、16×16 多種都能整除檔案、測試也能通過的候選。
最後靠初始化常數與原版畫面才定案。每個 graphics decoder 都必須輸出 atlas 給人看；
「成功解出 N 格」不是視覺驗收。

### decompiler 可能安靜地編造控制流

16-bit real-mode 的間接 jump table 若沒有正確 override，decompiler 不一定報錯，
反而可能輸出貌似合理的錯誤 C。重要 switch 必須讀原始指令與 table words；必要時
用 IDA 的 code discovery、Ghidra override 與直接 byte decoding 交叉驗證。

### 欄位解出來卻沒接 runtime，仍然是 bug

`MONSTER.DAT` 第七數值欄早已證明是 armour points，但 Go 欄位仍叫
`NumAttacks`，建立怪物時也完全沒寫入 `Unit.Armor`。文件完成不等於功能完成；
每個 resolved field 都要檢查「parser → typed field → runtime copy → consumer → test」。

### 待辦清單也會腐化

`MONSTER.DAT special`、吐息素材等項目在深層筆記已解，總 worklist 卻仍列為未知。
每次逆向結案都應全 repo 搜尋舊說法，同步改 worklist、格式文件、程式註解與測試。

### 原版與現代化必須分開命名

EGA、CGA 是兩套原版素材，不是高低畫質；Modern EGA 是第三套可選重畫，也不能稱為
「原版還原」。theme identity 應與 source decode mode 分離，完整預載後原子切換。

### 抽樣測試要抽規則類型，不只抽地點

機制與資料驅動串接確定後，可以用攻略抽驗後期；但樣本要覆蓋不同狀態轉換：
購船、密語、符印、頭目、結局、存檔覆寫，而不是只挑幾個相距很遠的房間。

## 共用 skill 的安裝位置

- Codex：`~/.codex/skills/reverse-engineer-retro-game-remake`
- GitHub skill 工作目錄：`~/my_skill/reverse-engineer-retro-game-remake`

skill 本體保持通用，不包含《冬之魔》的私有路徑、固定位址或遊戲專屬規則；這些案例
留在本專案文件中。共用 skill 只保留可跨專案重現的判斷方法、模板與驗收閘門。

