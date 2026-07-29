package scenario

import (
	"bytes"
	"fmt"
	"os"
)

// 以下常數描述 PARTY.DAT 的整體版面（已驗證，見 docs/formats/game-data-tables.md §1.1）：
//
//	0x000 - 0x103   1 號角色記錄（260 bytes）
//	0x104 - 0x207   2 號角色記錄
//	0x208 - 0x30b   3 號角色記錄
//	0x30c - 0x40f   4 號角色記錄
//	0x410 - 0x513   5 號角色記錄
//	0x514 - 0x5d5   隊伍共用資料（194 bytes，trailer）
const (
	// recordLen 是單一角色記錄的長度，用 5 個角色姓名字串在檔案內的起始位址
	// 互減驗證過（0x000/0x104/0x208/0x30c/0x410，差值全部是 0x104）。
	recordLen = 0x104
	// numCharacters 是隊伍固定人數。
	numCharacters = 5
	// nameFieldLen 是姓名欄位保留的長度（NUL 結尾字串）。
	nameFieldLen = 12
	// inventoryStart/inventorySlotLen/inventorySlotCount 是裝備／道具欄。
	//
	// **起點是 0x0c，不是先前記載的 0x1a**（見 docs/formats/game-data-tables.md）。
	// 兩者都能讓 10 個 slot 剛好接到 EXP 欄位（0x1a + 170 = 0xc4），所以
	// 「邊界對得上」不足以區分；定案依據是 DOSBox 實驗中改動的四個 byte，
	// 只有 0x0c 起算能全部解釋。
	//
	// 這個位移先前被誤當成種族欄位（見 raceOffset）。
	inventoryStart     = 0x0c
	inventorySlotLen   = 17
	inventorySlotCount = 10
	inventoryRegionLen = inventorySlotLen * inventorySlotCount // 170
	// expOffset 是經驗值欄位在角色記錄內的相對位移。
	//
	// **儲存寬度是 4 bytes**（0xc4–0xc7），數值封頂在 0x00FFFFFF ——
	// 先前記載的「3 bytes」是把封頂值當成了欄位寬度。技能旗標從 0xc8 起，
	// 正好接在 4 bytes 之後。
	expOffset = 0xc4
	expLen    = 4

	// skillFlagsOffset/skillFlagsLen 是已學技能旗標陣列，每項 1 byte（0/1），
	// 索引就是遊戲內部技能 id（見 docs/re/21-skills-races-and-files-dat.md）。
	skillFlagsOffset = 0xc8
	skillFlagsLen    = 31

	// 神殿相關的三個 1-byte 欄位（`docs/re/19` §8，全部已驗證）。
	//
	//	+0xeb  祈禱成功率，直接存百分比（0–20）。改宗與祈禱都寫成 20，
	//	       戰鬥中每次呼喚成功 −5。
	//	+0xec  束縛效果的等級。治療所的解束縛費用 = 它 × 費率，復活時清零。
	//	+0xf0  信奉的神祇編號。**0 代表沒有信仰**；非 0 時減一才是
	//	       神祇名表（FILES.DTT `[153:164]`）的索引，見 docs/re/27 §4。
	// identifiedTodayOffset 是「本日已研究過道具」的旗標（`1000:1ee9` 設 1，
	// 睡覺 `2aed:0513` 清 0）。與打獵的 `+0xef` 是同一組「每日一次」旗標。
	identifiedTodayOffset = 0xed

	// exorcisedTodayOffset 是「本日已驅邪」的旗標（`1000:19bd` 設 1）。
	// 與 `+0xed`（鑑定）、`+0xef`（打獵）同一組每人每天一次。
	// worshipedTodayOffset 是「本日已敬拜」的旗標（`1000:0f0c` 設 1）。
	worshipedTodayOffset = 0xf1

	exorcisedTodayOffset = 0xf2

	prayChanceOffset = 0xeb
	bindLevelOffset  = 0xec
	deityOffset      = 0xf0

	// levelOffset/raceOffset/classOffset 是三個 1-byte 欄位。
	//
	// 種族**在 0xf5**，不是先前記載的 0x0c —— 那個位移其實是道具槽陣列的起點。
	// 職業（0–9）是技能學費表的欄索引，兩個獨立呼叫點都這樣用。
	levelOffset = 0xf4
	raceOffset  = 0xf5
	classOffset = 0xf6
	// attrXxxOffset 全部是「相對於 expOffset 的位移」，沿用 docs/walkthrough
	// 與 tools/parse_party.py 已驗證的寫法，1 byte，不含加成/含加成分開存放。
	attrStrengthNaturalOffset = 0x24
	attrSkillNaturalOffset    = 0x25
	attrMaxSPNaturalOffset    = 0x26
	// attrSpeedNaturalOffset：已驗證且修正攻略筆誤——原攻略把 3 號角色這個位址
	// 標成「Max SP」，但依其他角色的欄位間距、以及與「速度（含加成）」欄位數值
	// 交叉比對（本存檔 5 名角色皆無影響速度的裝備，天生值應等於含加成值，實測
	// 確實相等；若真是法力值則應等於「法力值（含加成）」，實測不相等），
	// 確認這裡是速度，不是法力值。見 docs/formats/game-data-tables.md §1.4。
	attrSpeedNaturalOffset  = 0x2f
	attrSpeedBonusOffset    = 0x33
	attrStrengthBonusOffset = 0x34
	attrIntellectOffset     = 0x35
	attrEnduranceOffset     = 0x36
	attrSkillBonusOffset    = 0x37
	attrMaxHPOffset         = 0x38
	attrCurrentHPOffset     = 0x39
	attrMaxSPBonusOffset    = 0x3a
	attrCurrentSPOffset     = 0x3b

	// weaponSlotOffset/armorSlotOffset/combatFlagsOffset 是角色記錄尾端 4 bytes
	// 裡的前 3 個欄位（絕對位移，不是相對 expOffset）。**待複核**：反組譯
	// docs/re/06-combat-system.md 推得的欄位，尚未做 DOSBox 動態複核
	// （裝備已知道具、觸發已知戰鬥狀態後 diff 存檔驗證）。
	weaponSlotOffset  = 0x100
	armorSlotOffset   = 0x101
	combatFlagsOffset = 0x102
	unknown103Offset  = 0x103
)

// trailerStart 是隊伍共用資料（trailer）在檔案內的起始絕對位移。
const trailerStart = numCharacters * recordLen // 0x514

// trailerLen 是 trailer 的長度（194 bytes，0x514-0x5d5）。
const trailerLen = 194

// 以下常數是 trailer 內部的已知／待複核欄位位移，全部相對 trailerStart（0x514）。
const (
	// formationOffset/Len：**3×3 陣型格**（已驗證，見 `docs/re/34`）。
	// 九個 byte 各存一名成員的編號，0xFF 是空格。排法就是紮營選單
	// Reorder 畫面上的那張圖：
	//
	//	   A B C      cell 0 1 2
	//	   D E F      cell 3 4 5
	//	   G H I      cell 6 7 8
	//
	// 原本標「假設」，長度也猜成 10。四個呼叫端把它釘死了：Reorder
	// （`1000:0379` 先把 9 格清成 0xFF 再逐一填）、離隊（`0x14af2`
	// 清掉該員並把大於他的編號往前挪）、入隊（`0x15089` 找第一個空格）、
	// 佈陣（`0xc615` 把格號換算成相對座標）。四處的迴圈上界都是 9。
	//
	// PARTY.DAT 是 `00 ff 01 ff 02 ff 03 ff 04`（五人散在 A C E G I），
	// PARTY.BAK 是 `00 01 02 03 04 ff ff ff ff`（擠在前五格）—— 都合理。
	formationOffset = 0x00
	formationLen    = 9

	// reserved09Offset 是陣型格與金幣之間的一個保留 byte。IDA 9.4 全檔
	// consumer 稽核未找到 PARTY struct +0x09 的玩法讀寫；兩份原版存檔皆為
	// 0。仍原樣 round-trip，不能推論其他平台或版本也未使用（docs/re/112）。
	reserved09Offset = 0x09

	// goldOffset：隊伍金幣，**4 bytes（0x0a–0x0d）**。
	//
	// 這裡一度標成「長度未知」，理由是兩份存檔的第 4 個 byte 都是 0、
	// 分不出 3 還是 4。**分不出來就去找讀寫端** —— 買船的扣款
	// （`DEMON.INT 0x1FF5A`，`2aed:148a`）是一組 32-bit 減法：
	//
	//	sub word es:[bx+0x0a], ax
	//	sbb word es:[bx+0x0c], dx
	//
	// 跨 `+0x0a` 與 `+0x0c` 兩個 word，也就是 4 bytes。`docs/re/19` §2 早就
	// 把它列成 32-bit 了，是本檔沒跟上。數值上限比照經驗值封在 0x00FFFFFF。
	goldOffset = 0x0a
	goldLen    = 4

	// ldFlagsOffset/Len：**未知**。兩份存檔在這 7 個 byte 只出現 0x4c('L')
	// 或 0x44('D') 兩種值（DAT: L L L L D L D，BAK: L D D D D L D），是否真的是
	// ASCII 字母、還是恰好落在這個值域的旗標，未能判斷。
	ldFlagsOffset = 0x0e
	ldFlagsLen    = 7

	// partySizeOffset：**隊伍人數**（已驗證）。睡覺（`2aed:048f`）與紮營清旗標
	// 那幾段都拿它當「掃過每名隊員」的迴圈上界。
	partySizeOffset = 0x9a

	// rationsOffset：**糧食份數**（已驗證）。紮營睡覺會 −1（`2aed:057a`），
	// 打獵成功會 += 收穫並鉗在 255（`1000:0933`–`0945`）。
	rationsOffset = 0x9b

	// hourOffset：**時辰**（已驗證，反組譯）。0x164f7 對它 `inc`，同一段
	// 程式把 stepCounter（+0xa0）歸 1 —— 走滿一小時的步數就進位。
	// 戰場的視野內縮量拿它當索引查 DEMON.INT 的時辰表
	// （見 gamedata.LightInsetAt），所以它確實是「幾點」而不是別的計數。
	hourOffset = 0x9f

	// monthOffset／dayOffset：**月與日**（已驗證，反組譯）。睡覺常式
	// `0x1f1d0` 起那一段是完整的兩層進位：
	//
	//	inc [bx+0x9e] / cmp al,0x23 / jb skip / mov [bx+0x9e],1   ; 日，34 天一個月
	//	inc [bx+0x9d] / cmp al,0x17 / jb skip / mov [bx+0x9d],1   ; 月
	//
	// 狀態列（`0x70ac`）把 `+0x9d` 乘 4 當索引查 `ds:0x50f2` 的遠指標表，
	// 取出**月份名稱**再套進 `"Hour %d, Day %d in the Month of the %s"`。
	// 所以月是 0-based 的名稱索引 —— 原版存檔的 0 是合法值，不是未初始化。
	// 日則是 1-based：新遊戲初始化（`0x14908`）寫的是 8。
	monthOffset = 0x9d
	dayOffset   = 0x9e

	// timeCounterOffset：**一小時之內的步數計數**（已升級為已驗證）。
	// DOSBox 動態 diff 早就看到「移動時每步 +1」；反組譯補上了另一半 ——
	// 0x164ed 把它設回 1、同時把 hourOffset `inc`，所以它是進位到時辰的
	// 那個計數器，不是遊戲內時間本身。
	timeCounterOffset = 0xa0

	// positionXOffset/positionYOffset：**待複核**（反組譯/動態 diff 推得，
	// 高信心但單一樣本，尚未大量覆核）。DOSBox 實跑「往不同方向各走一步、
	// 存檔、diff」得出：僅左右移動時 X 變動、Down +1／Up -1 時 Y 變動。
	// Y 是螢幕座標，向下為正。見 docs/formats/game-data-tables.md §1.5
	// 「DOSBox 動態 diff 補充」。
	positionXOffset = 0xa1
	positionYOffset = 0xa2

	// mapIDOffset：**目前的子地圖編號**（已驗證，反組譯）。走到子地圖邊界時
	// 換圖的四段程式碼（DEMON.INT 0x16fec–0x17114）都在改它：往東西 ±10、
	// 往南北 ±1，因為編號 = 欄×10 + 列（見 world/grid.go）。
	// 別處拿 `>= 10` 當「在戶外還是在地城」判斷 —— 編號 10 以下是地城。
	mapIDOffset = 0xa3

	// lightSourceOffset：**地城的光源強度**（已驗證，反組譯）。
	// DEMON.INT 0x161ab 在「身處地城」時把它讀進 ds:0x5c64，
	// 戰場視野內縮量 = 4 − 它（0x161c5）。兩份原版存檔都是 1，
	// 換算成 3×3 的視野 —— 火把照得到的範圍，數值合理。
	lightSourceOffset = 0xa7

	// boatOffset：**目前搭乘的船**（已驗證，反組譯）。存的是船隻陣列的
	// 格號 **+1**，`0` 代表沒有搭船。走上有船的格子時 `0x16ede` 寫入
	// `格號+1`，上岸時 `0x16dd6` 清成 0。見 `docs/re/31`。
	// viewedLandTodayOffset 是「本日已用過觀地」的**隊伍層級**旗標
	// （`1000:09b6` 設 1，睡覺 `2aed:03f?` 清 0）。角色層級的每日旗標
	// 是 `+0xed`（鑑定）與 `+0xef`（打獵）—— 觀地一天只能用一次，
	// 而且是整隊共用一次。
	viewedLandTodayOffset = 0xac

	// formationBackupOffset：**試煉室用的 3×3 陣型備份**（`docs/re/101` §2）。
	//
	// 試煉室只讓符合職業的人上場，做法是把陣型整張搬到這裡（`0x0f34b`）、
	// 清空、只填回那幾個人；戰鬥收尾常式一開頭就搬回去（`0x0e01d`，
	// **勝敗都搬**）。不打的那幾條路當場就還原（`0x0f42c`／`0x0f6b9`）。
	//
	// 它必須進存檔而不是只放在記憶體：原版就放在 trailer 裡，
	// 而「陣型被清空」這個狀態如果漏掉還原，隊伍會變成只剩一個人上場。
	formationBackupOffset = 0x80

	// provingPassedOffset：**十間試煉室各一格「過了沒」**（`docs/re/101` §2.2）。
	//
	// 一個職業一間（Ranger…Scholar，索引與 `ds:0x17e1` 那張職業名表同序）。
	// 全檔只有三處碰它，全部是 `[bx+si+0x8a]`：進試煉室過關寫 1（`0x0f677`）、
	// 戰勝之後寫 1（`0x0e1d2`）、**地點劇情 case 8 數還有幾間沒過**（`0x1a29c`）。
	// 十間全過才拿得到恆世寶珠。
	provingPassedOffset = 0x8a
	// ProvingRoomCount 是試煉室的間數（也就是職業數）。
	ProvingRoomCount = 10

	// provingRoomOffset：**現在人在第幾間試煉室**，格號 **+1**，0 代表不在
	// （`docs/re/101` §2）。
	//
	// 它的存在只為了跨過一場戰鬥：進去時寫 `索引+1`，戰勝的收尾常式
	// （`0x0e1bc`）讀它決定要把 `+0x8a` 的哪一格標成過關，然後清 0。
	// **不打的那幾間（盜賊／靈視者／學者）當場就清掉。**
	provingRoomOffset = 0xab

	// poolDrinksOffset：**還能喝幾口治療水池**（`docs/re/90`）。
	//
	// 全檔只有三處碰它：水池比較是不是 0（`0x196ca`，空了就印
	// `The pool is empty`）、喝一次遞減（`0x1974a`）、**睡覺重設成 7**
	// （`0x1eee6`，就在印 `You sleep.` 的那支裡）。
	// 所以它是隊伍層級的每日次數，與觀地（`+0xac`）同一組，額度是 7 不是 1；
	// **換一口池子不會回滿**。
	poolDrinksOffset = 0xaa

	// viewRoomUsesOffset／viewItemUsesOffset：兩個靈視技能的**每日次數**
	// （`docs/re/93`）。上限都是 **3**（`0x19027`／`0x19422` 的 `cmp …,3`），
	// 到了就印 `Your psychic powers are weak`。睡覺與 `+0xac`（觀地）
	// 在同一段清 0（`0x1ef68`–`0x1ef7c`）。
	//
	// ⚠ 手冊說這兩個技能「每天只能使用一次」—— 那是 Apple II 版的說法，
	// DOS 版的執行檔寫的是 3。
	viewRoomUsesOffset = 0xad
	viewItemUsesOffset = 0xae

	// encounterCountdownOffset：**離下一場隨機戰鬥還有幾步**（`docs/re/51`）。
	//
	// 走一步減一，歸零時主迴圈回傳動作碼 `0x16`（`0x16aee`）去挑怪。
	// **不是機率是倒數** —— 這個 byte 一度被當成語意未解。
	// 兩份原版存檔是 56 與 34，正好落在「戰鬥後重設 28–77」的範圍內。
	encounterCountdownOffset = 0x9c

	// glyphFlagsOffset：**三個緋紅符印的劇情旗標**（`docs/re/59`）。
	//
	// 索引 ＝ 子地圖編號 − 55（子地圖 66 特判成 2），對應世界東南角
	// 那三塊狹長陸地。`0` ＝ 符印還在、`0x80` ＝ 已用 UNCURSE 解除。
	//
	// 這是本專案目前唯一的主線進度：三個都非 0，才進得了光之環
	// （`0x1a569`），否則被 crimson forcefield 擋下。符印還在時
	// 在附近走動會被 `FUN_222f_0619` 傷害。
	//
	// 原版對這三個 byte 只有讀，沒有以常數定址的寫入 ——
	// 寫入端用 `es:[bx+si+0x96]`，三格共用同一段程式碼（`docs/re/59` §1）。
	glyphFlagsOffset = 0x96
	glyphCount       = 3

	// merchantBaseOffset：**商隊規模的基準值**（`docs/re/50`）。
	// 進地圖時 `0x15f5d` 把它讀進 `ds:0x5c60`；在戶外（子地圖編號 > 9）
	// 會被地圖記錄自帶的參數覆蓋。商隊遭遇拿它算
	// `規模 = clamp(基準 + rnd(3) − 2, ≤ 9)`，而**規模同時就是商隊等級**。
	// 兩份原版存檔是 1 與 4 —— 所以早期遇到的都是小商隊。
	merchantBaseOffset = 0xaf

	// plotGiftOffset 是**劇情送的道具「已經拿過了」旗標陣列**的起點
	// （`docs/re/65` §3.2）。送道具那支常式（`25be:11ff` ＝ `0x1a9df`）
	// 在 `0x1aacc` 做 `party[0xb3 + param] = 1`，而地點劇情 case 4（鐵匠鋪）
	// 的入口閘門 `0x1a0fd` 檢查的正是 `+0xb7` ＝ `0xb3 + 4`。兩邊對上。
	//
	// ⚠ **只認 6 格（`+0xb3`–`+0xb8`），而且是刻意的。**
	// 那支常式的跳表有 7 個 param，param 6 會寫到 `+0xb9` ——
	// 而 `+0xb9` 是**劇情階段**（`plotStageOffset`，`docs/re/81` 驗過）。
	//
	// **2026-07-27 查清楚了（`docs/re/101` §3）：不是衝突，是原版刻意共用。**
	// param 6 的呼叫端是地點劇情 case 8（十間試煉室全過才給的恆世寶珠），
	// 而「拿到寶珠」正是劇情推進到階段 1 的那個事件 —— `docs/re/81` 從
	// 馬利馮預言的第一行 `The Orb of Evertime now is yours` 推測過寫入端在
	// 寶珠事件裡，就是這裡。**這也是 `docs/re/80` §3 那個「找不到 `+0xb9 = 1`
	// 寫入端」的答案**：那一處是 `mov es:[bx+si], 1`，基底是
	// 「`party + 0xb3` 存進區域遠指標」，disp 為 0，常數位移掃不到。
	//
	// **所以仍然不擴到 7 格**：`+0xb9` 已經有 `plotStageOffset` 這個名字與
	// 讀取端，再給它第二個名字才是真正的漂移。case 8 要直接推進劇情階段。
	plotGiftOffset = 0xb3
	// PlotGiftCount 是上面那個陣列的格數（見警告）。
	PlotGiftCount = 6

	boatOffset = 0xb0

	// templeRuinsOffset／shardShatteredOffset：**世界變成廢墟的兩個旗標**
	// （`docs/re/79`、`docs/re/83`）。冬之魔降臨的機制實作 —— 地圖上的設施會消失。
	//
	//   - `+0xba > 0x7f` → 繪製時把神殿 tile `0x25` **換成廢墟 tile `0x5b`**
	//     （`0x1739a`），而且踩上去印 `You are walking through ruins`。
	//     它被寫的唯一時機是 `+0xb9 == 2`（`0x03ed0`），**同一段程式碼
	//     還把全隊的薩滿與司祭技能清成 0**（`0x03eea`／`0x03efc`）——
	//     那是整支執行檔裡對這兩個技能旗標的**唯一寫入**。
	//     所以神殿被毀＝驅邪與祈禱永久失效。
	//   - `+0xbe` 原本被命名為「城鎮廢墟旗標」，那是**看效果不看成因**。
	//     11 處存取裡只有兩處寫入，而且都在艾瑞戈爾那一場
	//     （`0x1b2ef`／`0x1b459`，`docs/re/83` §3）—— 也就是馬利馮在黑鏡裡
	//     捏碎春之石的那一刻。所以這個欄位的身分是**「碎片已碎，冬天開始」**，
	//     城鎮變廢墟只是它的下游效果之一：
	//       * `0x19135`：城鎮 tile `0x2e` 走廢墟路徑，不再進城
	//       * `0x15f7b`／`0x15fb6`：繪製時把 tile > 9 換成 2
	//       * `0x1a550`：艾瑞戈爾那一格的入口閘門，談過就不再談
	//     它是單向閂鎖，沒有任何一處寫 0。
	//
	// 兩者都是**只增不減**：世界一旦壞掉就回不去了。
	templeRuinsOffset    = 0xba
	shardShatteredOffset = 0xbe

	// eregoreMetOffset：**艾瑞戈爾專用的一格狀態**（`docs/re/83` §3）。
	// 全檔只有三處存取，而且都在那一場裡（`0x1b2dd` 比較、`0x1b2e5` 清 0、
	// `0x1b4ca` 設 1）—— 不是通用旗標。
	// 語意是「上次談崩，打過一架」：立起來之後再去找他就跳過全部問答，
	// 直接播馬利馮那段結尾。
	eregoreMetOffset = 0x99

	// plotStageOffset／firstDreamOffset：**睡覺推進的劇情階段**（`docs/re/80`）。
	//
	// 冬之魔是在玩家睡覺的時候降臨的。`1000:0206` 每晚只走一段：
	//
	//	月份 > 3 且 +0xbd == 0   → +0xbd = 1，播夢境第 0 頁（馬利馮的警告）
	//	+0xb9 == 1               → +0xb9 = 2，播第 1 頁（冬之魔降臨）
	//	+0xb9 == 2 且 +0xba == 0 → 神殿全毀＋信仰歸零，播第 2 頁
	//
	// ⚠ **把 `+0xb9` 從 0 推到 1 的那道寫入還沒找到**（`docs/re/81`）——
	// 常數位移與計算式寫入兩種掃法都掃過。`+0xb9` 共 19 處存取，
	// 五處寫入寫的是 2 或 3，沒有一處寫 1。攻略說觸發點是拿到恆世寶珠
	// 之後，DOSBox 實跑也證實這條路徑活著，所以缺的是「誰把它設成 1」。
	// **艾瑞戈爾那一場也不是**（`docs/re/83` §3 逐指令看過，它只寫 `+0xbe`）。
	plotStageOffset  = 0xb9
	firstDreamOffset = 0xbd

	// TempleRuinsThreshold 是神殿旗標的判斷門檻（原版 `cmp 0x7f / jbe`）。
	// 注意它與城鎮旗標的門檻**不同**（那邊是 `!= 0`）——
	// 兩個旗標語意相近但比較方式不一樣，照原版分開寫，不要統一。
	TempleRuinsThreshold = 0x7f

	// endingOfferedOffset 是 trailer 的**最後一個 byte**：
	// **結局抉擇的一次性閂鎖**。
	//
	// 這個欄位一度叫 `UnknownC1`（「語意未解」）。`docs/re/95` §3.5 解開了：
	// 動作碼 `S1`（冰之祭壇 ＋ 祈禱卷軸）先看劇情階段 `+0xb9`，
	// 再看這個值 —— **不為 0 就直接返回**（`0x185de`），
	// 所以「Who wishes to become a god?」一輪遊戲只問一次。
	//
	// 另一個讀取端是紮營選單的 Drop：型別 0x1d 的道具要它不為 0
	// 才丟得掉（`1000:21fe`，`docs/re/33` §3）——「結局已經開場」
	// 之後才准丟，兩邊語意是一致的。起始存檔是 0。
	endingOfferedOffset = 0xc1

	// facingOffset：**待複核**，中高信心（單一角色樣本，反向確認未做）。
	// DOSBox 動態 diff 顯示對應順時針四方位，推測 0=北 1=東 2=南 3=西。
	facingOffset = 0xa4
)

// Character 是 PARTY.DAT 一名角色記錄解析後的乾淨表示。
//
// 驗證狀態分三級（詳見 docs/formats/game-data-tables.md §1.2-1.5）：
//   - 「已驗證」：攻略原文明講位址、或有交叉驗證證據（位址間距、DAT/BAK diff、
//     數值內容比對）。
//   - 「待複核」：反組譯／單次 DOSBox 動態 diff 推得，尚未大量樣本覆核，
//     欄位語意標在對應常數的註解裡。
//   - 未知欄位一律保留原始 bytes，不瞎猜命名（見 Unknown0D、InventorySlotsRaw）。
type Character struct {
	// Name 是角色姓名（已驗證）。
	Name string

	// Level 是角色等級（0xf4）。
	Level byte

	// RaceByte 是種族索引（0xf5）：0 人類、1 精靈、2 矮人、3 黑暗精靈、4 巨魔。
	// 對照已由 FILES.DAT 0x422 的種族上限表與手冊附錄 B 交叉驗證（25/25 全對）。
	RaceByte byte

	// ClassByte 是職業索引（0xf6，0–9）：0 遊俠、1 聖騎士、2 蠻族、3 武僧、
	// 4 牧師、5 盜賊、6 巫師、7 術士、8 靈視者、9 學者。
	// 這個值同時是技能學費表（FILES.DAT 0x158）的欄索引。
	ClassByte byte

	// SkillFlags 是 31 項技能的已學旗標，索引即遊戲內部技能 id
	// （0 劍擊 … 30 硬化皮膚，見 docs/re/21）。值為 1 表示已學。
	SkillFlags [skillFlagsLen]byte

	// Inventory 是 10 個裝備／道具欄的已解欄位。未解的部分仍在
	// InventorySlotsRaw 裡，兩者指向同一份資料。
	Inventory [inventorySlotCount]InventorySlot

	// InventorySlotsRaw 是 10 個裝備／道具欄位，每個 17 bytes，語意未解
	// （只驗證了 slot 邊界與數量），保留原始 bytes。
	InventorySlotsRaw [inventorySlotCount][]byte

	// Experience 是經驗值（4 bytes little-endian，數值封頂 0x00FFFFFF）。
	Experience int

	// 屬性區（已驗證，見 docs/formats/game-data-tables.md §1.2）：
	StrengthNatural byte // 力量（天生值，不含裝備加成）
	SkillNatural    byte // 技巧（天生值）
	MaxSPNatural    byte // 最大法力值（天生值）
	SpeedNatural    byte // 速度（天生值）。已驗證且修正攻略筆誤，見 attrSpeedNaturalOffset 註解
	SpeedBonus      byte // 速度（含裝備加成）
	StrengthBonus   byte // 力量（含裝備加成）
	Intellect       byte // 智力
	Endurance       byte // 耐力
	SkillBonus      byte // 技巧（含裝備加成）
	MaxHP           byte // 生命值上限
	CurrentHP       byte // 目前生命值
	MaxSPBonus      byte // 最大法力值（含裝備加成）
	CurrentSP       byte // 目前法力值

	// IdentifiedToday 是「本日已在營地研究過道具」（+0xed），睡一晚清掉。
	IdentifiedToday bool

	// WorshipedToday 是「本日已敬拜」（+0xf1）。
	WorshipedToday bool

	// ExorcisedToday 是「本日已驅邪」（+0xf2）。
	ExorcisedToday bool

	// PrayChance 是祈禱（呼喚神祇）的成功率百分比（+0xeb）。
	// BindLevel 是束縛效果的等級（+0xec），解除束縛的費用依它計價。
	// Deity 是信奉的神祇編號（+0xf0），0 代表沒有信仰。
	PrayChance byte
	BindLevel  byte
	Deity      byte

	// WeaponSlotIndex 是目前裝備的武器對應哪個道具槽。**待複核**（反組譯推得，
	// docs/re/06-combat-system.md，尚未動態複核）。
	WeaponSlotIndex byte
	// ArmorSlotIndex 是目前裝備的護甲對應哪個道具槽。**待複核**（同上）。
	ArmorSlotIndex byte
	// CombatStatus 是戰鬥狀態。**不是位元旗標，是單一列舉值** ——
	// 舊名 `CombatStatusFlags` 誤導，已正名（見 CONTEXT 的推翻清單）。
	CombatStatus CombatStatus
	// Unknown103 是角色記錄最後一個未知 byte，值不固定，語意未知。
	Unknown103 byte

	// Raw 是這名角色完整的原始 260 bytes，供未來 encode（寫回存檔）使用；
	// decode 已知欄位之外的部分（包含前述所有「未知」區段）都能從這裡取得。
	Raw []byte
}

// CombatStatus 是角色的戰鬥狀態（記錄 +0x102）。
//
// **是單一列舉值，不是位元旗標。** 專案一度把它當旗標，已推翻。
type CombatStatus byte

const (
	// StatusNormal 正常。
	StatusNormal CombatStatus = 0
	// StatusPoison 中毒。睡覺時會依睡眠時數扣血（見 game.Rest）。
	StatusPoison CombatStatus = 1
	// StatusBound1／2／3 是束縛的三個等級。
	StatusBound1 CombatStatus = 2
	StatusBound2 CombatStatus = 3
	StatusBound3 CombatStatus = 4
	// StatusDead 死亡。
	StatusDead CombatStatus = 5
)

// SaveGame 是 PARTY.DAT 解析後的乾淨表示：5 名角色 + 隊伍共用 trailer。
type SaveGame struct {
	// Characters 是隊伍的 5 名角色，順序與檔案內原始順序一致。
	Characters [numCharacters]Character

	// Formation 是 3×3 陣型格（見 formationOffset 註解）。
	Formation [formationLen]byte

	// Reserved09 是陣型格與金幣之間的保留 byte。此 DOS 執行檔沒有玩法
	// consumer；保留它是為了原版存檔的逐 byte round-trip（docs/re/112）。
	Reserved09 byte

	// Gold 是隊伍金幣（4 bytes little-endian，見 goldOffset 註解）。
	Gold int

	// LDFlags 是 7 bytes 的未知旗標陣列（見 ldFlagsOffset 註解，觀察值只有
	// 'L'(0x4c) 或 'D'(0x44)，是否真的是這兩個 ASCII 字母未能判斷）。
	LDFlags [ldFlagsLen]byte

	// PositionX/PositionY 是隊伍目前地圖座標。**待複核**（DOSBox 動態 diff
	// 推得，見 positionXOffset/positionYOffset 註解）。Y 是螢幕座標，向下為正。
	PositionX byte
	PositionY byte

	// Facing 是隊伍朝向。**待複核**（見 facingOffset 註解），推測
	// 0=北 1=東 2=南 3=西，單一樣本、反向確認未做。
	Facing byte

	// MapID 是目前所在的子地圖編號（已驗證，見 mapIDOffset 註解）。
	// 11–77 是世界地圖（編號 = 欄×10 + 列），10 以下是地城。
	//
	// 交叉驗證：兩份原版存檔分別是 1 與 3 —— 正好是現存三個獨立地城檔
	// （MAP1／MAP3／MAP5.MAP）裡的兩個。
	MapID byte

	// LightSource 是地城的光源強度（見 lightSourceOffset 註解）。
	LightSource byte

	// GlyphFlags 是三個緋紅符印的劇情旗標（見 glyphFlagsOffset 註解）。
	GlyphFlags [glyphCount]byte

	// TempleRuins 是神殿全毀旗標；ShardShattered 是「春之石已碎、
	// 冬天開始」（見 templeRuinsOffset 註解）。保留原始 byte 而不是 bool ——
	// 兩者的判斷門檻不同（`> 0x7f` vs `!= 0`），轉成 bool 會把差別磨掉。
	TempleRuins    byte
	ShardShattered byte

	// EregoreMet 是艾瑞戈爾的兩階段狀態（見 eregoreMetOffset 註解）。
	EregoreMet byte

	// PlotGifts 是「劇情送的那幾件道具拿過了沒」（見 plotGiftOffset 註解）。
	PlotGifts [PlotGiftCount]byte

	// PlotStage／FirstDream 是睡覺推進的劇情階段（見 plotStageOffset 註解）。
	PlotStage  byte
	FirstDream byte

	// MerchantBase 是商隊規模的基準值（見 merchantBaseOffset 註解）。
	MerchantBase byte

	// EncounterCountdown 是離下一場隨機戰鬥還有幾步。
	EncounterCountdown byte

	// PartySize 是隊伍人數（已驗證，見 partySizeOffset 註解）。
	PartySize byte
	// Rations 是糧食份數（已驗證，見 rationsOffset 註解）。
	Rations byte

	// ViewedLandToday 是「本日已用過觀地」（trailer +0xac，整隊共用）。
	ViewedLandToday bool

	// FormationBackup 是試煉室借走陣型時的備份（見 formationBackupOffset 註解）。
	FormationBackup [formationLen]byte

	// ProvingPassed 是十間試煉室各自過了沒（見 provingPassedOffset 註解）。
	ProvingPassed [ProvingRoomCount]byte
	// ProvingRoom 是現在人在第幾間試煉室，**格號 +1**，0 代表不在
	// （見 provingRoomOffset 註解）。
	ProvingRoom byte

	// PoolDrinks 是今天還能喝幾口治療水池（trailer +0xaa，整隊共用）。
	PoolDrinks byte

	// ViewRoomUses／ViewItemUses 是兩個靈視技能今天用過幾次
	// （trailer +0xad／+0xae，上限 3）。
	ViewRoomUses byte
	ViewItemUses byte

	// Boat 是目前搭乘的船（船隻陣列格號 +1，0 代表沒搭船）。
	// 見 boatOffset 註解與 ship.go。
	Boat byte

	// EndingOffered 是結局抉擇的一次性閂鎖（見 endingOfferedOffset 註解）。
	EndingOffered byte

	// Hour 是遊戲內時辰（已驗證，見 hourOffset 註解）。
	Hour byte

	// Day 是日（1-based，34 天進一個月）、Month 是月（**0-based**，
	// 直接當月份名稱表的索引）。見 dayOffset／monthOffset 註解。
	Day   byte
	Month byte

	// TimeCounter 是疑似遊戲內時間／回合計數的候選欄位。**待複核**，中信心
	// （見 timeCounterOffset 註解）。
	TimeCounter byte

	// Ships 是世界上的 10 艘船（trailer +0x22，見 ship.go）。
	// **不是「我的船」** —— 原版沒有記載歸屬，修船只看船在不在腳邊。
	Ships [shipCount]Ship

	// TrailerRaw 是完整的 194 bytes trailer 原始內容，供未來 encode 使用，
	// 也涵蓋前述具名欄位之外、完全未解的 trailer 區域。
	TrailerRaw [trailerLen]byte
}

// LoadSaveGame 解析指定路徑的 PARTY.DAT（或 PARTY.BAK），回傳 5 名角色與
// 隊伍共用資料。
//
// 只做讀取（decode）；寫入在 saveenc.go（`Encode`／`SaveTo`），驗收標準是
// 「讀出來再寫回去 byte-for-byte 相同」。仍有「待複核」的欄位（金幣寬度、
// 隊形順序表）採保守策略：只覆蓋已解出的部分，其餘 bytes 原封不動。
//
// 呼叫端負責傳入檔案路徑；本函式一律用唯讀方式開檔（os.ReadFile），不會寫入
// 任何檔案，PARTY.DAT 這類原版存檔可放心以唯讀路徑傳入。
func LoadSaveGame(path string) (*SaveGame, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("scenario: 讀取 %s 失敗: %w", path, err)
	}

	minLen := trailerStart + trailerLen
	if len(data) < minLen {
		return nil, fmt.Errorf(
			"scenario: %s 長度 %d bytes 小於預期的 %d bytes（5 名角色記錄 + trailer）",
			path, len(data), minLen,
		)
	}

	var save SaveGame
	for i := 0; i < numCharacters; i++ {
		rec := data[i*recordLen : (i+1)*recordLen]
		ch, err := parseCharacter(rec)
		if err != nil {
			return nil, fmt.Errorf("scenario: 解析 %s 第 %d 號角色失敗: %w", path, i+1, err)
		}
		save.Characters[i] = ch
	}

	trailer := data[trailerStart : trailerStart+trailerLen]
	copy(save.TrailerRaw[:], trailer)
	copy(save.Formation[:], trailer[formationOffset:formationOffset+formationLen])
	save.Reserved09 = trailer[reserved09Offset]
	save.Gold = le4(trailer[goldOffset : goldOffset+goldLen])
	copy(save.LDFlags[:], trailer[ldFlagsOffset:ldFlagsOffset+ldFlagsLen])
	save.MapID = trailer[mapIDOffset]
	save.LightSource = trailer[lightSourceOffset]
	save.MerchantBase = trailer[merchantBaseOffset]
	copy(save.GlyphFlags[:], trailer[glyphFlagsOffset:glyphFlagsOffset+glyphCount])
	save.TempleRuins = trailer[templeRuinsOffset]
	save.ShardShattered = trailer[shardShatteredOffset]
	save.EregoreMet = trailer[eregoreMetOffset]
	copy(save.PlotGifts[:], trailer[plotGiftOffset:plotGiftOffset+PlotGiftCount])
	save.PlotStage = trailer[plotStageOffset]
	save.FirstDream = trailer[firstDreamOffset]
	save.EncounterCountdown = trailer[encounterCountdownOffset]
	save.PartySize = trailer[partySizeOffset]
	save.Rations = trailer[rationsOffset]
	save.Hour = trailer[hourOffset]
	save.Day = trailer[dayOffset]
	save.Month = trailer[monthOffset]
	save.PositionX = trailer[positionXOffset]
	save.PositionY = trailer[positionYOffset]
	save.Facing = trailer[facingOffset]
	save.TimeCounter = trailer[timeCounterOffset]
	save.Ships = parseShips(trailer)
	save.ViewedLandToday = trailer[viewedLandTodayOffset] != 0
	copy(save.FormationBackup[:], trailer[formationBackupOffset:formationBackupOffset+formationLen])
	copy(save.ProvingPassed[:], trailer[provingPassedOffset:provingPassedOffset+ProvingRoomCount])
	save.ProvingRoom = trailer[provingRoomOffset]
	save.PoolDrinks = trailer[poolDrinksOffset]
	save.ViewRoomUses = trailer[viewRoomUsesOffset]
	save.ViewItemUses = trailer[viewItemUsesOffset]
	save.Boat = trailer[boatOffset]
	save.EndingOffered = trailer[endingOfferedOffset]

	return &save, nil
}

func parseCharacter(rec []byte) (Character, error) {
	if len(rec) != recordLen {
		return Character{}, fmt.Errorf("角色記錄長度 %d 不是預期的 %d bytes", len(rec), recordLen)
	}

	name := rec[0:nameFieldLen]
	if idx := bytes.IndexByte(name, 0x00); idx >= 0 {
		name = name[:idx]
	}

	var slots [inventorySlotCount][]byte
	for i := 0; i < inventorySlotCount; i++ {
		s := inventoryStart + i*inventorySlotLen
		slot := make([]byte, inventorySlotLen)
		copy(slot, rec[s:s+inventorySlotLen])
		slots[i] = slot
	}

	var skills [skillFlagsLen]byte
	copy(skills[:], rec[skillFlagsOffset:skillFlagsOffset+skillFlagsLen])

	raw := make([]byte, recordLen)
	copy(raw, rec)

	attr := func(off int) byte { return rec[expOffset+off] }

	return Character{
		Name:              string(name),
		Level:             rec[levelOffset],
		RaceByte:          rec[raceOffset],
		ClassByte:         rec[classOffset],
		SkillFlags:        skills,
		InventorySlotsRaw: slots,
		Inventory:         parseInventory(slots),
		Experience:        le4(rec[expOffset : expOffset+expLen]),
		StrengthNatural:   attr(attrStrengthNaturalOffset),
		SkillNatural:      attr(attrSkillNaturalOffset),
		MaxSPNatural:      attr(attrMaxSPNaturalOffset),
		SpeedNatural:      attr(attrSpeedNaturalOffset),
		SpeedBonus:        attr(attrSpeedBonusOffset),
		StrengthBonus:     attr(attrStrengthBonusOffset),
		Intellect:         attr(attrIntellectOffset),
		Endurance:         attr(attrEnduranceOffset),
		SkillBonus:        attr(attrSkillBonusOffset),
		MaxHP:             attr(attrMaxHPOffset),
		CurrentHP:         attr(attrCurrentHPOffset),
		MaxSPBonus:        attr(attrMaxSPBonusOffset),
		CurrentSP:         attr(attrCurrentSPOffset),
		IdentifiedToday:   rec[identifiedTodayOffset] != 0,
		WorshipedToday:    rec[worshipedTodayOffset] != 0,
		ExorcisedToday:    rec[exorcisedTodayOffset] != 0,
		PrayChance:        rec[prayChanceOffset],
		BindLevel:         rec[bindLevelOffset],
		Deity:             rec[deityOffset],
		WeaponSlotIndex:   rec[weaponSlotOffset],
		ArmorSlotIndex:    rec[armorSlotOffset],
		CombatStatus:      CombatStatus(rec[combatFlagsOffset]),
		Unknown103:        rec[unknown103Offset],
		Raw:               raw,
	}, nil
}

// le4 把 4 bytes 讀成 little-endian 無號整數（經驗值與金幣）。
func le4(b []byte) int {
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
}

// parseInventory 解出 10 格道具的已知欄位。
func parseInventory(slots [inventorySlotCount][]byte) [inventorySlotCount]InventorySlot {
	var out [inventorySlotCount]InventorySlot
	for i, raw := range slots {
		out[i] = parseInventorySlot(raw)
	}
	return out
}
