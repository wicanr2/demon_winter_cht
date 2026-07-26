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
	// formationOrderOffset/Len：**假設**「隊形/順序表」。PARTY.DAT 內容
	// 00 ff 01 ff 02 ff 03 ff 04 00、PARTY.BAK 是 00 01 02 03 04 ff ff ff ff 00，
	// 兩者都出現角色索引 0-4 並伴隨大量 0xFF，支持「這是隊形/順序表、
	// 0xFF=空位」的假設，但排列方式的實際意義未解。
	formationOrderOffset = 0x00
	formationOrderLen    = 10

	// goldOffset：隊伍金幣。**長度未知**——攻略只說「和經驗值一樣反序儲存」，
	// 沒講長度；PARTY.DAT 與 PARTY.BAK 在這個欄位後兩個 byte 剛好都是 0x00，
	// 無法從現有兩份存檔判斷究竟是 3 byte（類比經驗值）還是標準 4 byte long。
	// 這裡固定讀 3 bytes（與經驗值欄位一致的假設），第 4 byte 保留在
	// TrailerRaw 供之後比對，不在此處採信任一種猜測。
	goldOffset = 0x0a

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

	// unknown9COffset：**待複核**，低信心。DOSBox 動態 diff 顯示移動時每步 -1，
	// 與 timeCounterOffset 反向連動，候選是口糧／體力，未對到遊戲內 UI 確認。
	unknown9COffset = 0x9c

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

	// FormationOrder 是隊形/順序表（**假設**，見 formationOrderOffset 註解）。
	FormationOrder [formationOrderLen]byte

	// GoldRaw3 是隊伍金幣的 3-byte little-endian 讀值（**長度未知**，見
	// goldOffset 註解）。呼叫端若要嘗試 4-byte 讀法，可自行從 TrailerRaw
	// 在 goldOffset 位置多讀 1 byte。
	GoldRaw3 int

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

	// PartySize 是隊伍人數（已驗證，見 partySizeOffset 註解）。
	PartySize byte
	// Rations 是糧食份數（已驗證，見 rationsOffset 註解）。
	Rations byte

	// Hour 是遊戲內時辰（已驗證，見 hourOffset 註解）。
	Hour byte

	// Day 是日（1-based，34 天進一個月）、Month 是月（**0-based**，
	// 直接當月份名稱表的索引）。見 dayOffset／monthOffset 註解。
	Day   byte
	Month byte

	// TimeCounter 是疑似遊戲內時間／回合計數的候選欄位。**待複核**，中信心
	// （見 timeCounterOffset 註解）。
	TimeCounter byte

	// Unknown9C 是與 TimeCounter 反向連動（每步 -1）的候選欄位，疑似口糧／
	// 體力。**待複核**，低信心（見 unknown9COffset 註解）。
	Unknown9C byte

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
	copy(save.FormationOrder[:], trailer[formationOrderOffset:formationOrderOffset+formationOrderLen])
	save.GoldRaw3 = le3(trailer[goldOffset : goldOffset+3])
	copy(save.LDFlags[:], trailer[ldFlagsOffset:ldFlagsOffset+ldFlagsLen])
	save.MapID = trailer[mapIDOffset]
	save.LightSource = trailer[lightSourceOffset]
	save.PartySize = trailer[partySizeOffset]
	save.Rations = trailer[rationsOffset]
	save.Hour = trailer[hourOffset]
	save.Day = trailer[dayOffset]
	save.Month = trailer[monthOffset]
	save.PositionX = trailer[positionXOffset]
	save.PositionY = trailer[positionYOffset]
	save.Facing = trailer[facingOffset]
	save.TimeCounter = trailer[timeCounterOffset]
	save.Unknown9C = trailer[unknown9COffset]

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

// le3 把 3 bytes 讀成 little-endian 無號整數（trailer 的金幣欄位；
// 該欄位實際寬度未定，見 goldOffset 註解）。
func le3(b []byte) int {
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16
}

// le4 把 4 bytes 讀成 little-endian 無號整數（角色經驗值欄位）。
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
