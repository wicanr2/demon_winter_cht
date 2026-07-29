# 《冬之魔》重製引擎抽離可行性研究

> 日期：2026-07-29  
> 範圍：只依本 repository 的程式碼、資料格式文件與反組譯成果判斷。  
> 結論中的「已證實」表示已有程式碼、位元組或反組譯證據；未取得其他遊戲檔案前，
> 不把同公司、同年代或同系列當成共用引擎的證據。

## 結論

**值得抽離，但現在只適合做「可抽離化」，不適合立刻發布成通用遊戲引擎。**

建議先在同一個 Go module 內切出穩定邊界，讓《冬之魔》繼續當第一個、也是目前唯一
通過驗證的 consumer。等第二款遊戲通過本文末尾的相容性檢查、實際接入後，再決定是否
建立獨立 module 或 repository。

原因如下：

- 已有一批明顯可重用、且與劇情無關的能力：CGA/EGA 圖片與 sprite 解碼、點陣字、
  64×64 地圖容器、確定性 RNG、輸入/畫面/音效框架、資料來源與存檔覆寫優先序。
- 現行程式已經自然分成 `internal/assets/*`、`internal/game`、`internal/ui` 與
  `cmd/demonwinter`，不是從單一巨大檔案起步。
- 但原版的 `.INT` **不是 bytecode 或腳本 VM**，而是有 3,807 筆 relocation 的
  MS-DOS MZ 原生 8086 程式，由 loader 用 DOS `EXEC` 啟動。事件資料只提供文字、
  怪物與少數參數；事件條件、劇情推進、城鎮服務、職業、法術與戰鬥規則大量寫死在
  `DEMON.INT`。這表示目前沒有一套已證實可跨作品載入的「SSI 腳本引擎」可以抽出。
- `cmd/demonwinter` 的 `app` 同時持有地圖、存檔、各種 modal UI、戰鬥、海戰、
  劇情文字、城鎮、音效與除錯狀態。若現在直接搬成公共 package，只會把
  《冬之魔》的耦合公開化，之後難以維持相容性。

因此，推薦目標不是「把 `internal/game` 改名成 engine」，而是：

> 先抽出經第二個 consumer 也可能需要的機制；保留《冬之魔》的資料 schema、
> 規則與流程為 game adapter。第二款作品用真實資料接入後，再由兩個實例歸納 API。

## 證據與限制

### 已證實的原版架構

| 層次 | repository 證據 | 對抽離的意義 |
|---|---|---|
| 啟動 | `docs/research/ssi-engine-architecture.md`：`DEMON.EXE` loader 以 DOS `EXEC` 啟動 MZ 格式的 `DEMON.INT` | 沒有可抽離的 bytecode VM |
| 流程 | `docs/formats/event-script.md`、`internal/assets/scenario/events.go`：`DATA*.TXT` 是 NUL 分隔的固定欄位記錄 | parser 可重用，但欄位 grammar 是否跨遊戲共用仍待驗證 |
| 圖形 | `docs/formats/graphics.md`、`internal/assets/gfx`：CGA/EGA `.PIC/.PIE/.SHP/.SHE/.FNT/.FNE` 已解碼並與原版畫面比對 | 是目前最成熟的可抽離候選 |
| 世界 | `internal/assets/world`：64×64 map、`SUM.MAP`、獨立 `MAPn.MAP`、城鎮與出口資料 | grid/container 可抽；檔名與 map-id 規則不能假設通用 |
| 遊戲表格 | `internal/assets/gamedata`：`FILES.DAT/DTT`、怪物、道具、城鎮等 parser | parser 與 domain 型別目前混合，先保留在遊戲 adapter |
| 狀態 | `internal/assets/scenario/save.go`：角色、隊伍、船與進度的固定 offset schema | binary primitives 可抽；`PARTY.DAT` schema 是《冬之魔》專屬，除非第二款逐欄吻合 |
| 規則 | `internal/game` 的戰鬥、法術、神祇、海戰、事件 gate、劇情與經濟 | 多數是《冬之魔》規則，不應包裝成「通用」 |
| 執行期 | `cmd/demonwinter/main.go` 的 `app` | 可抽 state stack/render loop；現有 screen state 必須先解耦 |

### 目前不能證明的事

- 不能從「SSI 出品」推出 Gold Box 或其他 SSI 遊戲使用同一引擎。
- 不能從副檔名同為 `.INT` 推出檔案是相同 VM；本作的 `.INT` 已證實只是原生 MZ
  executable。
- 《冬之魔》是 *Shard of Spring* 的續作，這只提供「優先驗證候選」，
  **不構成共用程式碼或資料格式證據**。
- `Demon's Winter Town Maker` 的 Applesoft BASIC 殘留證實 Apple II 工具鏈與資料移植，
  但不能證明另一作品使用相同 Town Maker 或相同 record layout。
- Novotrade 負責 IBM PC 移植的證據，反而提醒我們：DOS port 的 loader、視訊層或資料轉換
  可能是外包移植專屬，不能直接外推到 SSI 自製 DOS 作品。

## 建議的 package boundary

第一階段維持單一 module，避免尚未穩定的 API 變成外部承諾。建議的依賴方向：

```text
cmd/demonwinter
    │
    ├── games/demonwinter/runtime   畫面流程、screen/controller 組裝
    ├── games/demonwinter/rules     現有戰鬥、法術、城鎮、劇情、經濟
    └── games/demonwinter/data      PARTY/FILES/DATA/TOWN 等專屬 schema
                 │
                 ▼
          engine/runtime            loop、screen stack、input action、clock
          engine/render             logical canvas、palette、tiles/sprites/font
          engine/grid               bounds、direction、grid/path primitives
          engine/storage            data source、overlay save、atomic write
          engine/random             可注入、可重播的 RNG interface
                 │
                 ▼
          formats/legacy/ssi        經跨作品驗證後才放進來的格式
```

`engine/*` 不得 import `games/demonwinter/*`。遊戲層可 import 引擎層，並以 adapter
滿足小型 interface。

### 建議的最小 API

```go
type Screen interface {
    Update(Input) (Transition, error)
    Draw(Canvas)
}

type TileMap interface {
    Bounds() image.Rectangle
    TileAt(x, y int) (uint16, bool)
}

type TileSet interface {
    Tile(index int) Image
    NativeTileSize() image.Point
}

type DataSource interface {
    Open(name string) (io.ReadCloser, error)
}

type SaveStore interface {
    Load(slot string) ([]byte, error)
    Commit(slot string, data []byte) error
}

type Random interface {
    Intn(n int) int
}
```

重點不是上述名稱，而是限制：

- `Screen` 不知道怪物、法術或城鎮。
- renderer 不知道 Eregore、Demon Crystal、海戰或劇情旗標。
- storage 不寫死 `DEM_DATA`、`PARTY.DAT` 或 `MAP%d.MAP`。
- grid 不依賴 `gamedata.Tables`；可通行與成本由遊戲 adapter 提供。
- RNG 不直接建立全域種子，測試與錄影可重播同一序列。

### 不應先抽的 API

- `Battle`, `Spell`, `TownVisit`, `Worship`, `PlotGift`, `SeaBattle`
- `FILES.DAT` 的 skill/race/summon table
- `PARTY.DAT` 的角色、船、劇情 trailer layout
- `DATA*.TXT` 的 continuation/rune/monster 語意
- tile index 對應的城鎮、海面、門、陷阱與劇情行為

這些名稱雖然看似常見，現有實作的欄位和值域都來自《冬之魔》反組譯。它們可以在
`games/demonwinter` 內保持乾淨與可測試，但在第二款作品吻合前不應成為引擎契約。

## 分階段方案

### Phase 0：建立抽離基線

成本：低；風險：低。

1. 保持目前測試、原版截圖與視覺 dump 為基準。
2. 記錄 package import graph、啟動/存檔/戰鬥的 golden path。
3. 為相同 seed、save 與輸入序列建立 deterministic replay smoke test。
4. 把現有 EGA/CGA F8 切換當成 renderer 不改規則狀態的驗證案例。

驗收：重構前後 save bytes、重要規則測試與基準截圖不變。

### Phase 1：先抽低爭議基礎元件

成本：中；風險：低至中。

- 移動 `internal/assets/gfx` 的純解碼與 palette 邏輯到格式/渲染邊界；
  Ebiten image 包裝留在 renderer。
- 抽出 `DataSource`、save overlay 與安全寫入。
- 抽出 grid direction/bounds，不搬《冬之魔》的 passability table。
- 抽出 logical canvas、nearest-neighbor scaling、點陣字排版與 PC speaker transport。
- 保留相容 wrapper，避免一次改動所有呼叫端。

驗收：`games/demonwinter` 是唯一 consumer，但不再需要 renderer 知道資料檔名或劇情型別。

### Phase 2：拆 `app` 的流程控制

成本：中至高；風險：中。

- 把 `app` 的大量 nullable modal 欄位改成 screen/state stack。
- 將資源載入組裝成 `DemonWinterContent`，把世界/戰鬥/城鎮 controller 分開。
- 輸入先轉成 semantic action，再交給畫面；鍵位與遊戲規則解耦。
- renderer 只消費 view model，不直接修改 save 或 battle。

驗收：可用 headless controller 跑開局、移動、事件、戰鬥與存檔，不建立 Ebiten window。

### Phase 3：用第二款真實遊戲做相容性 spike

成本：中（只做調查）到很高（完整支援）；風險：最高。

優先檢查 *Shard of Spring*，理由只有「同系列前作」，不是已證實相容。調查時：

1. 只取使用者合法持有的檔案做 hash、檔名、大小與 header inventory。
2. 檢查 loader、主檔是否也是 MZ + DOS `EXEC`，不要被 `.INT` 名稱誤導。
3. 用既有 decoder 試跑，但禁止以「能整除」當驗證；每類素材都要肉眼比對。
4. 比較 map、town、monster、item、party、event 的 record stride 與 consumer code。
5. 寫一個最小 adapter：開場圖 + 一張 map + 一個可移動角色；不要先移植完整規則。

只有第二款遊戲讓既有抽象不需要加入產品名稱分支，才進 Phase 4。

### Phase 4：決定是否獨立發布

成本：中；風險：中。

達標條件：

- 至少兩款遊戲使用相同 package，而非複製後各自改。
- 公共 package 不含 `if game == ...` 或產品檔名常數。
- 格式相容性有 fixture/hash、解析測試與視覺 oracle。
- 存檔/素材的授權界線清楚；engine repository 不夾帶原版資料。

達標後才考慮獨立 module。否則保留 monorepo 內的 internal framework，收益已足夠，
也沒有公共 API 維護負擔。

## 成本與收益

| 項目 | 收益 | 成本／代價 | 建議 |
|---|---|---|---|
| gfx decoder | 可供工具、remake、素材比對與第二款 adapter 共用 | 要分離純像素資料與 Ebiten 型別 | 優先 |
| screen/runtime | 減少 `app` nullable state，利於測試與錄影 | 觸及大量 UI flow | 分批 |
| storage/data source | 保護原版資料、統一 save overlay | 要整理既有路徑優先序 | 優先 |
| grid primitives | 地圖工具與遊戲共用 | 很容易誤把 tile 語意一起抽走 | 只抽幾何 |
| generic RPG combat | 表面上複用度高 | 本作規則、欄位與原版常數高度耦合 | 暫緩 |
| generic event engine | 看似可資料驅動 | 原版沒有 VM，流程多在 native code | 不做 |
| 獨立 repository | 可供外部專案引用 | API、版本、授權與 CI 維護負擔 | 第二款接入後再做 |

整體預估：Phase 0–1 是可控制的中型重構；Phase 2 是高觸及重構；Phase 3 的成本取決於
第二款格式相似度，不能在取得檔案與反組譯證據前可靠估算。最大風險不是 Go 搬 package，
而是把「一款遊戲的巧合」誤寫成長期公共契約。

## 風險

1. **過早泛化**：把 `FILES.DAT`、tile id 或戰鬥欄位包成 generic 名稱，實際仍是產品專屬。
2. **視覺假陽性**：本專案已有多次「檔案大小、frame 數與測試都通過，但方向/尺寸仍錯」
   的紀錄；相容性必須包含實圖比對。
3. **移植版差異**：Apple II 原作資料、Novotrade DOS port 與其他平台可能共享內容但不共享 runtime。
4. **god object 搬家**：直接把 `app` 放進 engine 不會降低耦合。
5. **存檔破壞**：抽 storage 時若改了 byte-for-byte round trip 或路徑優先序，可能覆蓋合法原版資料。
6. **效能抽象稅**：逐 tile/interface call 可能影響舊式軟體渲染；先量測，不預先設計複雜 cache。
7. **授權與散布**：decoder 和 engine 可開源，不代表原版資料、字型或美術能隨 engine 散布。
8. **研究範圍膨脹**：跨遊戲研究可能拖慢本作收尾；Phase 3 應設成有明確停止條件的 spike。

## 判定另一款遊戲是否相容的 checklist

每項記錄 `相同 / 可由參數調整 / 需要 adapter / 不相容 / 未知`，並附檔案 hash、
offset、反組譯地址或截圖，不只寫主觀結論。

### 執行檔與工具鏈

- [ ] loader 與主程式各自的格式、MZ header、relocation 數已確認。
- [ ] 已確認是 DOS `EXEC`、overlay、VM 或其他載入方式；不依副檔名猜測。
- [ ] 編譯器/runtime signature、x87 依賴與 platform port 身分已確認。
- [ ] 若聲稱 loader 同源，已有 binary/function-level similarity 證據。

### 圖形與音效

- [ ] `.PIC/.PIE/.SHP/.SHE/.FNT/.FNE` 的 header、stride、plane order、frame size 相同。
- [ ] palette 是同一編碼，並確認是否有執行期 palette override。
- [ ] 至少各一張全螢幕圖、portrait、tile、sprite 與字型經原版畫面肉眼驗證。
- [ ] PC speaker 資料與播放規則相同，或可由獨立 adapter 提供。
- [ ] EGA/CGA 差異可用 renderer theme 表達，不需要遊戲規則分支。

### 地圖與世界

- [ ] map 尺寸、header、tile width 與排列方向相同。
- [ ] `SUM.MAP` 分段方法和獨立 `MAPn.MAP` 規則相同。
- [ ] passability/terrain 是資料表還是程式常數已追到讀取端。
- [ ] town、exit、special tile record stride 與座標系相同。
- [ ] map 變更是否寫回存檔、覆寫優先序是否相同。

### 資料與存檔

- [ ] monster/item/town/string table 的 delimiter、欄位數和值域相同。
- [ ] event record grammar 已由 parser 消耗與原程式讀取端雙重驗證。
- [ ] party/character/item slot/ship record stride 逐欄比對，而非只比總長度。
- [ ] 未初始化 buffer/編輯器殘留已與有效資料區分。
- [ ] round-trip 不改未知 byte；原版能讀回測試存檔。

### 規則與流程

- [ ] movement、time、encounter gate 的狀態與順序相同。
- [ ] combat unit schema、AP、命中、傷害、AI 與 spell dispatch 相同。
- [ ] class/race/skill/deity/economy 表的維度與索引相同。
- [ ] 劇情條件位於資料、script 或 native code 的位置已確認。
- [ ] 不相同的規則能完全留在 game adapter，無需污染 `engine/*`。

### 抽象品質

- [ ] 第二款可使用相同 engine API，不需產品名稱條件分支。
- [ ] engine 測試不讀任何原版 copyrighted asset。
- [ ] 兩款遊戲各有自己的 fixture/hash 與 screenshot oracle。
- [ ] headless replay、save round-trip、EGA/CGA render smoke 都通過。
- [ ] 移除任一款遊戲後 engine package 仍能獨立編譯與測試。

## 最終建議

短期排程應採 **Phase 0 → Phase 1 的小步抽離**，並把 Phase 2 排在本作視覺、
逆向與玩法收尾之後。另開一個有時間盒的 *Shard of Spring* 相容性調查是合理的，
但在取得直接證據前，文件與 package 命名都應使用「Demon's Winter family candidate」
或中性的 legacy runtime，而不是宣稱「SSI engine」。

這條路能先取得測試性、工具共用與維護收益，同時避免為一套尚未證實存在的跨作品引擎
支付過早泛化的成本。
