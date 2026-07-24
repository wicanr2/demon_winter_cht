# Glossary 彙整紀錄 — 2026-07-24

彙整 11 份譯稿（`docs/manual/part-0.md`..`part-4.md`、`docs/walkthrough/part-1.md`..`part-6.md`）
末尾「新增譯名建議」節，併入 `translations/glossary.md`。本檔記錄「改了什麼」，供專案主持人核實。

## 總覽

- 新增約 **148 筆**譯名條目（含新增章節內的所有列）。
- 新增 **7 個章節**：第 15 節（陷阱類型）、16 節（地形）、17 節（怪物／NPC）、
  18 節（魔法道具類別與附魔機制）、19 節（戰鬥與遊戲機制用語）、20 節（系統選單／開場畫面）、
  21 節（文件與雜項用語）。編號接續第 14 節之後，**不插入原有章節之間**——因為
  `docs/manual/part-0.md`、`docs/walkthrough/part-2/3/5/6.md` 的既有文字裡直接寫死
  「glossary 第 X 節」引用原編號，插入式重編號會讓這些引用全部錯位。
- **沒有刪除或更動任何既有條目的翻譯本身**。有 3 筆既有條目被加註（未改變譯名）：
  - `The Ancient One`：加註與 `the Ancients`（新增）的區分說明。
  - `Detect aura` / `Dark vision`：分類欄從「種族」補充為「種族（DOS 版）」，並列出對應的
    Apple II 版新增條目。
  另有 2 個表格（第 8 節「附魔材質／品質詞」、第 11 節「主要地點」）因新增條目需要第 3 欄
  （備註），既有列被加上空白的第 3 欄以維持 Markdown 表格欄數一致——文字內容本身未變動。
- 第 10 節城鎮表補齊到 25 座（見下方「城鎮表驗證」）。
- 有 1 處對既有「待辦提示」文字加了刪除線並更新為完成狀態（見下方「例外處理」），
  是本次彙整中唯一一處touched既有非表格文字的地方。

## 逐檔來源對照

| 來源檔 | 貢獻筆數（約） | 主要內容 |
|---|---|---|
| `docs/manual/part-0.md` | 4 | Chants、Rule Book、Lore 泛稱、Caveat Emptor |
| `docs/manual/part-1.md` | 17 | 開場選單、移動指令、Sense magic/See in dark、Rune Magic、神祇陣營統稱、League、Kobold/Goblin |
| `docs/manual/part-2.md` | 24 | 地形 7 項、地形生物 10 項、College/Marketplace/Deity Call/Caveat Emptor |
| `docs/manual/part-3.md` | 27 | 陷阱 7 項、怪物 13 項、戰鬥／航海介面 12 項（含重複） |
| `docs/manual/part-4.md` | 32 | 附魔機制三分類、道具類別 11 項、武器潛伏能力 10 項、Brolor、kudzu |
| `docs/walkthrough/part-1.md` | 13 | 章節標題群、Inn/Ship、Andrew Schultz |
| `docs/walkthrough/part-2.md` | 4 | Sense magic/See in dark（重複確認）、hack、save or die |
| `docs/walkthrough/part-3.md` | 4 | save scumming、save or die/no save、Land's Edge（待確認）、kudzu area（待確認） |
| `docs/walkthrough/part-4.md` | 26 | 地點 7 項、NPC／怪物 12 項、道具 6 項、附魔效果、glyph/the Ancients/river Styx |
| `docs/walkthrough/part-5.md` | 26 | 附魔機制細節、道具類別、武器特殊力量 9 項 |
| `docs/walkthrough/part-6.md` | 6 | 附魔類型（另一版本）、Life Stealing（另一版本）、Pirate's Cove |

（部分詞在多檔重複出現，例如 Sense magic、Berserker、Mithril 等；上表按「該檔有提到」計，
不是去重後的獨立貢獻數，去重後的實收錄數見上方「總覽」148 筆。）

## 城鎮表驗證（第 10 節）

用 Python 標準庫解出 `workplace/orig/demwin/DEM_DATA/TOWN.TXT`：

```python
data = open('TOWN.TXT', 'rb').read()
parts = data.split(b'\x00')
```

前 25 個以 NUL 分隔的字串依序對應第 1–25 座城鎮，與 `TOWN1.DAT`..`TOWN25.DAT`
共 25 個檔案的數量一致（第 26 個字串起是城鎮傳聞旁白文字，不是城鎮名）。

- **第 24 座 = `Land's Edge`**，第 25 座 = `Pirate's Cove`。這直接解答了兩位譯者標的
  「待確認」——**兩者都是正式城鎮名**，不是攻略作者的隨口簡稱或待考證地名。
  - `Pirate's Cove` → 「海盜灣」：walkthrough/part-6 正文已多處使用「海盜灣」，本次確認為
    正式城鎮名後直接收錄，無需更動既有用字。
  - `Land's Edge` → **譯名本身仍待確認**：walkthrough/part-3 提到此地名時保留原文未譯，
    只在「新增譯名建議」節備註「音譯『蘭德邊境』或保留原文，待更多脈絡再定案」。
    現在已確認是正式城鎮（脈絡問題解決了），但**中文譯名還沒有人選定**，我在表格中列了
    兩個候選（蘭德邊境／天涯鎮）供主持人裁定，**沒有自己拍板一個填進去冒充定案**。
- 副產品：`TOWN.TXT` 第 25 個之後的字串是城鎮傳聞／劇情旁白，其中一則提到
  「Kobolds 的營地由一個叫 `Uffspgot` 的角色率領」——這是一個 11 份譯稿都沒提到的 NPC 名，
  **本次沒有收錄**（超出彙整範圍，且沒有既定譯名可依循），僅在此記錄供之後翻譯時參考。

## 已知衝突裁決（任務指定的 7 項）

| 衝突 | 裁決 | 理由 |
|---|---|---|
| `Sense magic`/`See in dark`（Apple II）vs `Detect aura`/`Dark vision`（DOS） | 兩版都收錄（第 5 節），互相加註對應關係 | 手冊譯文用 Apple II 用語、遊戲內文字用 DOS 用語，兩邊都要能查到，不是翻譯分歧 |
| `See in dark` 譯名本身 | 「黑暗視物」（manual/part-1 建議「夜視」列為備案） | 與 DOS 版「黑暗視覺」同字根，一望即知是同一能力的兩個版本用語 |
| `Berserking`（技能，狂暴）vs `Berserker`（武器潛伏能力，狂戰士） | 兩條都收，互相加註「勿混淆」 | 確認兩位譯者的區分正確——狂暴是可學技能，狂戰士是武器隨機觸發的附魔效果 |
| `Dwarven`（附魔材質／潛伏能力）vs `Dwarf`（種族矮人） | 兩條都收，互相加註「勿混淆」；`Dwarven` 定譯「矮人打造」 | 確認區分正確；「矮人打造」比部分譯者建議的裸詞「矮人」更能避免與種族名完全同字的歧義 |
| `save or die`/`no save`（D&D 豁免檢定術語） | 收錄，明確標註「非存檔，勿與 save/load 混淆」 | 譯者已提醒，直接照辦；另外發現 walkthrough/part-2、part-3 已發布正文各自用了不同措辭（見下方「額外發現」） |
| `glyph`（符印）vs `rune`（符文） | 兩詞分開收錄，互相標註區分 | 確認區分正確；`TOWN.TXT` 城鎮旁白文字提到「三個緋紅符印」，確認 glyph 在遊戲中是實際存在的獨立物件，不是泛稱 |
| `the Ancients`（複數，遠古種族）vs `The Ancient One`（單數，遠古者） | 兩詞分開收錄，互相標註區分 | 確認區分正確；`TOWN.TXT` 城鎮旁白文字兩處使用複數集合語境（"With the Ancients gone…"、"the Ancients became angry"），佐證複數用法成立 |
| `Chants` | 收錄「吟唱」，不再更動 | 專案主持人已裁定，直接照辦 |

## 額外發現的衝突（彙整過程中自己找到，非譯者已標出）

這部分比處理已知衝突更花時間，但也更有價值——找出來的都是「同一英文詞在不同檔被譯成不同中文」，
若不處理，後續遊戲內文字中文化時譯者會各自選一個，越滾越亂。

| 詞 | 各檔譯法 | 裁決 | 理由 |
|---|---|---|---|
| `Invoked (Powers)` | manual/4：賦予能力（**正文已用**）／walkthrough/4：喚醒／walkthrough/5：賦能力量（**正文已用**）／walkthrough/6：觸發（**正文大量已用**） | 觸發（能力） | 四方衝突，最難的一個。「觸發」明確傳達「玩家主動使用指令觸發」，「喚醒」易與「潛伏」的沉睡／喚醒聯想混淆，「賦予」「賦能」偏向授予語感、不傳達觸發動作。manual/part-4、walkthrough/part-5 兩篇已發布正文的既有用字**不在本次修改範圍內**，維持原樣 |
| `Constant (Powers)` | manual/4、walkthrough/4：恆常（能力）／walkthrough/6：常駐（**正文大量已用**） | 恆常（能力） | 與「潛伏」「觸發」並列維持同語感風格；walkthrough/6 既有正文用字不動，僅記錄分歧 |
| `Dormant Powers` | manual/3：潛藏能力／manual/4、walkthrough/4/5/6：潛伏（能力） | 潛伏（能力） | 多數且與各檔正文用字一致 |
| `Special Power(s)` | walkthrough/5：特殊力量（**正文已用**）／walkthrough/6：特殊能力（**正文已用**） | 特殊能力 | 與「恆常能力／潛伏能力／觸發能力」統一用「能力」字尾 |
| `Vorpal` | manual/4：斬殺／walkthrough/5：斬首 | 斬首 | D&D 系列「vorpal sword」通行中譯即「斬首劍」，源自《愛麗絲鏡中奇緣》vorpal blade 典故 |
| `Stasis` | manual/4：靜止／walkthrough/5：靜滯 | 靜滯 | 較能傳達「被定住／懸置」的魔法狀態感，不只是單純的「停止」 |
| `Life Stealing` | manual/4、walkthrough/5：奪命／walkthrough/6：汲取生命（呼應既有 Power leech＝汲取法力） | 奪命 | 與同組其他潛伏能力譯名（秘銀／邪惡／斬首／鋒銳／狂戰士／靜滯／詛咒）同為短促 2 字效果標籤，風格一致 |
| `Timber wolf` | manual/2：灰狼／manual/3：林狼 | 林狼 | 與姊妹生物「Winter wolf＝寒冬狼」構成一致的「棲地＋狼」命名家族，「灰狼」是真實物種俗名，與棲地命名邏輯無關 |
| `Rod` | manual/4：短杖／walkthrough/5：權杖 | 權杖 | 「短杖」與同表「Staff＝法杖」「Wand＝魔杖」字形字義太接近；「權杖」是 D&D 系奇幻譯名中 Rod 類道具的通行譯法 |
| `Crown` | manual/4：王冠／walkthrough/5：皇冠 | 王冠 | 語感較中性泛用，不特指「帝王」等級 |
| `Plus`／`Plus Weapons` | manual/3：附加武器／manual/4：加成道具 | 統一詞根「加成」（加成道具／加成武器） | 兩篇對同一「Plus」詞根用字不一致，改成同一詞根 |
| `Caveat Emptor` | manual/0（目錄提及）：購物風險自負／manual/2（章節本文標題）：買者自慎 | 買者自慎 | 採用章節本文自己的標題用字，比目錄的意譯轉述更貼近原標題簡潔語感 |
| `Deity Call` | manual/2（章節標題）：神祇呼喚／manual/3（描述戰鬥選單機制）：呼喚神明 | 神祇呼喚 | 同一機制的不同語境敘述，統一採章節標題慣用的名詞短語風格 |
| `save or die` | walkthrough/2 正文：一擊必殺（豁免失敗即死）／walkthrough/3 正文：抵抗失敗即死 | 豁免失敗即死（新標準） | 兩篇**已發布正文**用字已經不同，這是彙整前就存在、沒人發現的既成分裂。改用 D&D 標準術語「豁免檢定」而非「抵抗檢定」。**兩篇既有正文本次不動**，此為未來新內容的標準用法 |

## 待確認條目一覽

不假裝已定案的條目，全部照原譯者建議收錄並在備註標明「待確認」：

1. **`Land's Edge`**（第 10 節，城鎮 #24）：正式城鎮身分已用 `TOWN.TXT` 確認，但中文譯名沒有共識，
   表格暫記「待確認」，備註列出候選（蘭德邊境／天涯鎮）。
2. **`kudzu area`**（第 16 節）：是否為正式地名（相對於「葛藤地」這種通用地形描述）待後續章節確認，
   先照收「葛藤地帶」。
3. **`Brolor`**（第 17 節 NPC）：是矮人鍛造工坊的地名還是人名，原文脈絡不夠明確，待確認。

## 邊界與未做的事

- 只修改了 `translations/glossary.md` 與新建 `translations/glossary-changelog.md`，未動
  `docs/`、`PLAN.md`、`README.md`、`tools/`、`docker/`，也沒有動任何譯稿末尾的「新增譯名建議」節。
- 沒有 git commit / push。
- 上面標「額外發現」表中列出的既有正文用字分歧（`Constant`/`Invoked`/`Special Power`/
  `save or die` 等在不同章節已經用了不同中文），**glossary 只設定「未來標準用法」，
  沒有回頭改動任何一篇已發布的譯稿正文**——那些修改超出本次任務範圍（任務邊界明確禁止改
  `docs/`），是否要回頭統一，留給專案主持人決定。

## 一處小小的規則字面爭議，請主持人裁示

第 10 節原本有一行提示「> 剩餘城鎮名待從 `TOWN.TXT` 完整解出後補齊。」——這是一句「還沒做完」
的待辦提示，不是譯名條目本身。既然本次已經把 24、25 座都補上了，如果原封不動留著這句話，
會讓下一個讀這份表的人誤以為城鎮表還沒補齊（違反「code/文件是唯一真相，別留 stale marker」
的一般原則）。

我的處理方式：**沒有刪除這一行**，而是在原文字上加了刪除線（`~~...~~`）並在後面加註完成情形，
逐字保留原句供追溯。這技術上仍是「加註」而非「刪改」，但如果主持人認為連刪除線都不該碰、
應該完全原封不動保留，請告知，我可以改成「在該行前面另插一行完成提示、原句完全不動一個字」。
