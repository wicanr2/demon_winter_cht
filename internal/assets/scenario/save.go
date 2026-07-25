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
	// raceOffset 是種族欄位在角色記錄內的相對位移。0xFF = 尚未設定種族
	// （PARTY.BAK，角色創建早期存檔全部是 0xFF；PARTY.DAT 創建完成後是 0-4）。
	raceOffset = 0x0c
	// unknown0DOffset/unknown0DLen 是種族之後、裝備欄之前的未知區段，本次分析的
	// 存檔裡 5 名角色全部是 0，懷疑放職業或其他角色創建旗標，無法從這份存檔驗證。
	unknown0DOffset = 0x0d
	unknown0DLen    = 13
	// inventoryStart/inventorySlotLen/inventorySlotCount 是裝備／道具欄，
	// 結構（slot 邊界、數量）已用 0xFF/0x0a 出現位置的週期性 + PARTY.DAT vs
	// PARTY.BAK diff 交叉驗證過；slot 內部欄位語意未解，保留原始 bytes。
	inventoryStart     = 0x1a
	inventorySlotLen   = 17
	inventorySlotCount = 10
	inventoryRegionLen = inventorySlotLen * inventorySlotCount // 170
	// expOffset 是經驗值欄位（3 bytes little-endian）在角色記錄內的相對位移。
	// 攻略原文已明講位址；本次分析用 5 名角色位址間距交叉驗證過位置無誤。
	expOffset = 0xc4

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

	// unknown9COffset：**待複核**，低信心。DOSBox 動態 diff 顯示移動時每步 -1，
	// 與 timeCounterOffset 反向連動，候選是口糧／體力，未對到遊戲內 UI 確認。
	unknown9COffset = 0x9c

	// timeCounterOffset：**待複核**，中信心。DOSBox 動態 diff 顯示移動時每步
	// +1，候選是遊戲內時間／回合計數，尚未對到遊戲內實際顯示的時/日/月。
	timeCounterOffset = 0xa0

	// positionXOffset/positionYOffset：**待複核**（反組譯/動態 diff 推得，
	// 高信心但單一樣本，尚未大量覆核）。DOSBox 實跑「往不同方向各走一步、
	// 存檔、diff」得出：僅左右移動時 X 變動、Down +1／Up -1 時 Y 變動。
	// Y 是螢幕座標，向下為正。見 docs/formats/game-data-tables.md §1.5
	// 「DOSBox 動態 diff 補充」。
	positionXOffset = 0xa1
	positionYOffset = 0xa2

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

	// RaceByte 是種族索引原始值。0xFF = 尚未設定（角色創建早期存檔）。
	// **待複核**：索引 0-4 對應到哪個種族名稱是假設（依手冊列表順序
	// 人類0/精靈1/矮人2/黑暗精靈3/巨魔4），本存檔只出現過 0-2，
	// 索引 3、4 完全未經驗證。索引到名稱的對照刻意不在本套件內建，
	// 呼叫端如需顯示名稱請自行依 docs/formats/game-data-tables.md §1.2 對照，
	// 避免把未驗證的猜測寫死成「看似已驗證」的具名欄位。
	RaceByte byte

	// Unknown0D 是種族之後、裝備欄之前的 13 bytes，語意未知（保留原始值）。
	Unknown0D []byte

	// InventorySlotsRaw 是 10 個裝備／道具欄位，每個 17 bytes，語意未解
	// （只驗證了 slot 邊界與數量），保留原始 bytes。
	InventorySlotsRaw [inventorySlotCount][]byte

	// Experience 是經驗值（已驗證，3 bytes little-endian）。
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

	// WeaponSlotIndex 是目前裝備的武器對應哪個道具槽。**待複核**（反組譯推得，
	// docs/re/06-combat-system.md，尚未動態複核）。
	WeaponSlotIndex byte
	// ArmorSlotIndex 是目前裝備的護甲對應哪個道具槽。**待複核**（同上）。
	ArmorSlotIndex byte
	// CombatStatusFlags 是戰鬥狀態位元旗標（如中毒）。**待複核**（同上）。
	CombatStatusFlags byte
	// Unknown103 是角色記錄最後一個未知 byte，值不固定，語意未知。
	Unknown103 byte

	// Raw 是這名角色完整的原始 260 bytes，供未來 encode（寫回存檔）使用；
	// decode 已知欄位之外的部分（包含前述所有「未知」區段）都能從這裡取得。
	Raw []byte
}

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
// 只做讀取（decode）。寫入（encode，讓存檔能與原版互通）需要更多 DOSBox 動態
// 驗證（見本檔多處「待複核」欄位），這一輪不做——TODO(encode): 補齊裝備欄、
// 隊形順序表、金幣長度等待複核欄位的動態驗證後，再實作 SaveGame -> []byte
// 的編碼路徑，並用「讀出來再寫回去 byte-for-byte 相同」當最低驗收標準。
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

	unknown0D := make([]byte, unknown0DLen)
	copy(unknown0D, rec[unknown0DOffset:unknown0DOffset+unknown0DLen])

	raw := make([]byte, recordLen)
	copy(raw, rec)

	attr := func(off int) byte { return rec[expOffset+off] }

	return Character{
		Name:              string(name),
		RaceByte:          rec[raceOffset],
		Unknown0D:         unknown0D,
		InventorySlotsRaw: slots,
		Experience:        le3(rec[expOffset : expOffset+3]),
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
		WeaponSlotIndex:   rec[weaponSlotOffset],
		ArmorSlotIndex:    rec[armorSlotOffset],
		CombatStatusFlags: rec[combatFlagsOffset],
		Unknown103:        rec[unknown103Offset],
		Raw:               raw,
	}, nil
}

// le3 把 3 bytes 讀成 little-endian 無號整數（PARTY.DAT 經驗值／金幣欄位的
// 共用格式，攻略稱「反序」）。
func le3(b []byte) int {
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16
}
