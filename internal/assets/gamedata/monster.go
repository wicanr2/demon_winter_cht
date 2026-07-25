package gamedata

import "fmt"

// monsterTokensPerRecord 是每隻怪物在 MONSTER.DAT 裡佔用的 NUL 分隔 token 數：
// 1 個名字字串 + 11 個數字欄位。3853 bytes 的原始檔案切成 1188 個 token，
// 1188 ÷ 12 = 99 整除無餘數，對應 99 隻怪物，這個關係已用真實檔案驗證過
// （見 docs/formats/game-data-tables.md 第 2 節）。
const monsterTokensPerRecord = 12

// Monster 是 MONSTER.DAT 一筆怪物記錄解析後的乾淨表示。
//
// 驗證狀態（詳見 docs/formats/game-data-tables.md 第 2.2 節）分兩級：
//   - 「已驗證」：有交叉比對證據（跨等級數列、與 DEMON.INT 字串表對照、
//     同外觀種系數值一致等），可信賴其語意。
//   - 「語意未確認」：數值規律合理、也給了具名欄位，但沒有直接證據排除
//     其他解讀，未來若反組譯 DEMON.EXE 或動態測試有新結論要回來更新。
type Monster struct {
	// Name 是怪物名稱（例如 "Kobold"、"Cave bear"）。
	Name string

	// Speed 速度。已驗證：數值分布（4–26）與 PARTY.DAT 角色的速度屬性同量級，
	// 風元素（Wind elemental）速度全表最高，符合「風系最快」的直覺。
	Speed int
	// Strength 力量。已驗證：巨人/龍/惡魔類力量遠高於一般人形怪物，量級與
	// PC 種族力量上限相容。
	Strength int
	// Skill 技巧。已驗證：量級與 PC 屬性一致。
	Skill int
	// HP 生命值。語意未確認：「戰士」序列（Lvl 1 → Lvl 15 fighter）此欄位
	// 單調遞增（8,14,18,...,55），量級合理，但沒有直接證據排除這其實是
	// 「耐力」而非直接儲存的生命值上限。
	HP int
	// AttackType 攻擊／武器類型索引，負值＝帶毒。已驗證：對照 DEMON.INT
	// 的武器類型清單（索引 0–13，13=Bite），所有蛇類怪物此欄位全部是
	// 對應值的負版本（如 -13），精準對應「毒咬」機制，99 隻怪物無例外。
	AttackType int
	// SpriteIndex 圖像索引，對應 MONSTER.SHP/MONSTER.SHE。已驗證：同外觀
	// 種系怪物數值完全一致（4 種熊皆為 7、7 種龍皆為 27、6 種蛇皆為 12 等），
	// 只有「圖像索引」能合理解釋這種按外觀分組的規律。
	SpriteIndex int
	// NumAttacks 每回合攻擊次數。語意未確認：戰士序列此欄位隨等級遞增後
	// 趨緩（1,2,3,4,4,4,5,6,6），符合直覺但沒有戰鬥公式佐證。
	NumAttacks int
	// Experience 擊殺經驗值。已驗證：戰士序列單調遞增，終盤劇情怪物
	// （Xeres/Jesric/Eregore/Guardian）明顯高出一般怪物一個量級，且
	// DEMON.INT 有對應的戰鬥結算字串（"Exp per chr: %ld"）佐證機制存在。
	Experience int
	// Level 怪物等級／難度分級（1–10）。語意未確認：不等於名稱裡的數字
	// （例如 "Lvl 6 fighter" 此欄位是 5），推測是與名稱脫鉤的另一套內部
	// 難度分級，用途（遭遇表分級？法術抗性判定？）未定。
	Level int
	// SP 法力值上限。已驗證：「法師」序列（Lvl 1 mage → Lvl 16 wizard）
	// 單調遞增，非法師系怪物大多是 0，與 PARTY.DAT 的法力值屬性語意一致。
	SP int
	// Special 特殊能力／元素索引，-1＝無。語意未確認：龍系怪物（Fire/Wind/
	// Ice dragon）此欄位連續遞增（8,9,10，疑似吐息屬性索引），但元素系
	// 怪物（Fire/Metal/Wind/Ice/Spirit elemental）的值（1,2,3,7,5）不連續，
	// 對不上「五大符文系順序」的假設，整體規律不夠乾淨。
	Special int
}

// MonsterTable 是 MONSTER.DAT 解析後的查詢介面。
//
// 建立方式一律透過 LoadMonsterTable，零值不可用。
type MonsterTable struct {
	monsters []Monster
	byName   map[string]int
}

// LoadMonsterTable 解析指定路徑的 MONSTER.DAT，回傳可查詢的怪物表。
//
// 解析失敗（檔案讀不到、token 數對不上 12 的倍數、數字欄位不是合法整數）
// 一律回傳 error，不 panic。
func LoadMonsterTable(path string) (*MonsterTable, error) {
	tokens, err := readNULDelimitedTokens(path)
	if err != nil {
		return nil, err
	}
	if len(tokens)%monsterTokensPerRecord != 0 {
		return nil, fmt.Errorf(
			"gamedata: %s 的 token 數 %d 不是 %d 的倍數，MONSTER.DAT 格式假設可能不適用於這個檔案",
			path, len(tokens), monsterTokensPerRecord,
		)
	}

	n := len(tokens) / monsterTokensPerRecord
	monsters := make([]Monster, 0, n)
	byName := make(map[string]int, n)

	for i := 0; i < n; i++ {
		chunk := tokens[i*monsterTokensPerRecord : (i+1)*monsterTokensPerRecord]
		name := string(chunk[0])

		nums := make([]int, 0, monsterTokensPerRecord-1)
		fieldLabels := []string{
			"speed", "strength", "skill", "hp", "attack_type", "sprite_index",
			"num_attacks", "experience", "level", "sp", "special",
		}
		for j, label := range fieldLabels {
			v, err := parseASCIIInt(fmt.Sprintf("%s(monster #%d %s)", label, i, name), chunk[1+j])
			if err != nil {
				return nil, err
			}
			nums = append(nums, v)
		}

		m := Monster{
			Name:        name,
			Speed:       nums[0],
			Strength:    nums[1],
			Skill:       nums[2],
			HP:          nums[3],
			AttackType:  nums[4],
			SpriteIndex: nums[5],
			NumAttacks:  nums[6],
			Experience:  nums[7],
			Level:       nums[8],
			SP:          nums[9],
			Special:     nums[10],
		}
		monsters = append(monsters, m)
		byName[name] = i
	}

	return &MonsterTable{monsters: monsters, byName: byName}, nil
}

// Len 回傳怪物總數（MONSTER.DAT 目前是 99）。
func (t *MonsterTable) Len() int { return len(t.monsters) }

// All 回傳全部怪物的唯讀複本，順序與檔案內原始順序一致。
func (t *MonsterTable) All() []Monster {
	out := make([]Monster, len(t.monsters))
	copy(out, t.monsters)
	return out
}

// ByIndex 依索引（0-based，對應檔案內原始順序）取得怪物。
func (t *MonsterTable) ByIndex(i int) (Monster, error) {
	if i < 0 || i >= len(t.monsters) {
		return Monster{}, fmt.Errorf("gamedata: 怪物索引 %d 超出範圍 [0,%d)", i, len(t.monsters))
	}
	return t.monsters[i], nil
}

// ByName 依名稱取得怪物（完全比對，例如 "Cave bear"）。
func (t *MonsterTable) ByName(name string) (Monster, bool) {
	i, ok := t.byName[name]
	if !ok {
		return Monster{}, false
	}
	return t.monsters[i], true
}
