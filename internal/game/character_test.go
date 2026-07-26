package game

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func loadParty(t *testing.T) *scenario.SaveGame {
	t.Helper()
	p := filepath.Join(repoRoot(t), "workplace", "orig", "demwin", "DEM_DATA", "PARTY.DAT")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("game: 找不到 %s，略過需要真實資料的測試", p)
	}
	sg, err := scenario.LoadSaveGame(p)
	if err != nil {
		t.Fatalf("LoadSaveGame: %v", err)
	}
	return sg
}

// docs/spec/05-character.md 驗收 3：對這份存檔算出的剩餘智慧點數必須是
// Wopple 7、Stumpy 5、Podgom 9、Norman 9、Menhir 8。
//
// 這條同時把「技能 id 表、職業 id、學費表偏移、角色欄位偏移」四件事一起釘住 ——
// 任何一項判錯，數字就對不上。
func TestRemainingSkillPoints_MatchesSpecAcceptance(t *testing.T) {
	tb := loadTables(t)
	sg := loadParty(t)

	want := map[string]int{
		"Wopple": 7, "Stumpy": 5, "Podgom": 9, "Norman": 9, "Menhir": 8,
	}

	for _, sc := range sg.Characters {
		c := FromSave(sc)
		got, err := c.RemainingSkillPoints(tb)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		if w, ok := want[c.Name]; !ok {
			t.Errorf("出現預期外的角色 %q", c.Name)
		} else if got != w {
			t.Errorf("%s 剩餘智慧點數 = %d，預期 %d", c.Name, got, w)
		}
	}
}

func TestFromSave_MapsFields(t *testing.T) {
	sg := loadParty(t)

	// Wopple：精靈(1) 巫師(6) 等級 3，已學火焰符文(17) 與靈魂符文(21)。
	c := FromSave(sg.Characters[0])

	if c.Name != "Wopple" {
		t.Fatalf("姓名 = %q，預期 Wopple", c.Name)
	}
	if c.Race != gamedata.Elf {
		t.Errorf("種族 = %d，預期 %d（精靈）", c.Race, gamedata.Elf)
	}
	if c.Class != 6 {
		t.Errorf("職業 = %d，預期 6（巫師）", c.Class)
	}
	if c.Level != 3 {
		t.Errorf("等級 = %d，預期 3", c.Level)
	}
	if !c.HasSkill(gamedata.SkillFireRunes) || !c.HasSkill(gamedata.SkillSpiritRunes) {
		t.Error("應已學火焰符文與靈魂符文")
	}
	if c.HasSkill(gamedata.SkillSword) {
		t.Error("不應學過劍術")
	}
	if c.HasSkill(gamedata.SkillID(gamedata.NumSkills)) {
		t.Error("超出範圍的技能 id 應回傳 false")
	}
}

// 建角擲點：屬性不得低於下限；人類因為 +2 在鉗制之後，實際下限是 5。
func TestRollTraits_FloorAndHumanBonus(t *testing.T) {
	tb := loadTables(t)

	for _, tc := range []struct {
		race     gamedata.Race
		minValue int
	}{
		{gamedata.Human, traitFloor + humanTraitBonus},
		{gamedata.Elf, traitFloor},
		{gamedata.Troll, traitFloor},
	} {
		r := rng.NewWithSeed(1)
		for iter := 0; iter < 2000; iter++ {
			traits, err := RollTraits(r, tb, tc.race)
			if err != nil {
				t.Fatalf("RollTraits: %v", err)
			}
			for i, v := range traits {
				if v < tc.minValue {
					t.Fatalf("種族 %d 屬性 %d = %d，低於下限 %d", tc.race, i, v, tc.minValue)
				}
			}
		}
	}
}

// 基礎骰是 Roll(15) 這個假設，用值域來釘：修正為 0 的屬性，
// 觀察到的最大值必須剛好摸到 15，且不會超過。
//
// 刻意不用平均值來驗 —— 下限鉗制（<3 一律變 3）會把分佈往上拉，
// 算出來的平均不等於骰子的期望值 8，拿 8 當門檻只會驗到一個錯的東西。
func TestRollTraits_BaseDieRange(t *testing.T) {
	tb := loadTables(t)

	// 精靈的速度與技巧修正都是 0，可以直接看骰子的值域。
	for _, trait := range []gamedata.Trait{gamedata.Speed, gamedata.Skill} {
		mod, err := tb.RaceModifier(gamedata.Elf, trait)
		if err != nil {
			t.Fatalf("RaceModifier: %v", err)
		}
		if mod != 0 {
			t.Fatalf("精靈屬性 %d 的修正應為 0，得到 %d", trait, mod)
		}
	}

	r := rng.NewWithSeed(2024)
	lo, hi := math.MaxInt, math.MinInt
	for i := 0; i < 50000; i++ {
		traits, err := RollTraits(r, tb, gamedata.Elf)
		if err != nil {
			t.Fatalf("RollTraits: %v", err)
		}
		for _, trait := range []gamedata.Trait{gamedata.Speed, gamedata.Skill} {
			v := traits[trait]
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
		}
	}

	// 2d6 + 1 的上界。**原本這裡寫的是 Roll(15) 的上界 15** ——
	// 擲點公式改成與原版一致的 2d6 之後，上界是 13。
	if want := 2*traitDie + traitRollBonus; hi != want {
		t.Errorf("修正為 0 的屬性最大值 = %d，預期 %d（2d6+%d 的上界）",
			hi, want, traitRollBonus)
	}
	// 下界：2d6+1 最小是 3，剛好等於下限鉗制值，兩者在這裡分不開。
	if want := 2*1 + traitRollBonus; lo != want || lo != traitFloor {
		t.Errorf("修正為 0 的屬性最小值 = %d，預期 %d", lo, want)
	}
}

// docs/spec/05-character.md 驗收 4：耐力 17 的角色 N = 11，
// max(Roll(11), Roll(11)) 的期望值應趨近 7.818。
//
// 規格原本寫 7.7，那個數字算錯了：Roll(n) 回傳 1..n，兩次取大的期望值是
// Σ k(k²−(k−1)²)/n²，n=11 得 946/121 = 7.8182。規格已訂正。
func TestLevelUp_HPGainDistribution(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(999)

	const iters = 100000
	total := 0
	for i := 0; i < iters; i++ {
		c := Character{Race: gamedata.Human, Class: 0, MaxHP: 1, MaxSP: 1}
		c.Traits[gamedata.Endurance] = 17
		c.Traits[gamedata.Intellect] = 10
		// 屬性總和遠低於上限總和，分配不會被跳過。
		res, err := LevelUp(r, tb, &c)
		if err != nil {
			t.Fatalf("LevelUp: %v", err)
		}
		total += res.HPGain
	}

	got := float64(total) / float64(iters)
	const want = 946.0 / 121.0 // 7.8182
	if math.Abs(got-want) > 0.05 {
		t.Errorf("耐力 17 的 HP 成長平均 = %.4f，預期趨近 %.4f", got, want)
	}
}

// 升級不回血也不回魔。
func TestLevelUp_DoesNotHeal(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(7)

	c := Character{Race: gamedata.Human, MaxHP: 30, CurrentHP: 4, MaxSP: 20, CurrentSP: 2}
	c.Traits[gamedata.Endurance] = 12
	c.Traits[gamedata.Intellect] = 12

	if _, err := LevelUp(r, tb, &c); err != nil {
		t.Fatalf("LevelUp: %v", err)
	}

	if c.CurrentHP != 4 {
		t.Errorf("目前 HP = %d，升級不應改動（預期 4）", c.CurrentHP)
	}
	if c.CurrentSP != 2 {
		t.Errorf("目前 SP = %d，升級不應改動（預期 2）", c.CurrentSP)
	}
	if c.MaxHP <= 30 || c.MaxSP <= 20 {
		t.Errorf("最大值應成長，得到 HP %d / SP %d", c.MaxHP, c.MaxSP)
	}
}

// HP 封頂 255、SP 封頂 200，兩者不同。
func TestLevelUp_SeparateCaps(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(3)

	c := Character{Race: gamedata.Human, MaxHP: 254, MaxSP: 199}
	c.Traits[gamedata.Endurance] = 30
	c.Traits[gamedata.Intellect] = 30

	for i := 0; i < 20; i++ {
		if _, err := LevelUp(r, tb, &c); err != nil {
			t.Fatalf("LevelUp: %v", err)
		}
	}
	if c.MaxHP != maxHPCap {
		t.Errorf("最大 HP = %d，應封頂在 %d", c.MaxHP, maxHPCap)
	}
	if c.MaxSP != maxSPCap {
		t.Errorf("最大 SP = %d，應封頂在 %d", c.MaxSP, maxSPCap)
	}
}

// 屬性總和已達種族上限總和時，整個分配跳過 —— 否則「已滿則重骰」會無限迴圈。
func TestLevelUp_SkipsAllocationAtRacialCap(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(5)

	c := Character{Race: gamedata.Troll, MaxHP: 10, MaxSP: 10}
	for i := 0; i < gamedata.NumTraits; i++ {
		m, err := tb.RaceMax(gamedata.Troll, gamedata.Trait(i))
		if err != nil {
			t.Fatalf("RaceMax: %v", err)
		}
		c.Traits[i] = m
	}
	before := c.TraitSum()

	res, err := LevelUp(r, tb, &c)
	if err != nil {
		t.Fatalf("LevelUp: %v", err)
	}
	if !res.Skipped {
		t.Error("屬性已滿時應跳過分配")
	}
	if c.TraitSum() != before {
		t.Errorf("屬性總和 = %d，跳過分配時不應改變（預期 %d）", c.TraitSum(), before)
	}
}

// 分配一定是 3 點，且不得超過任何一項的種族上限。
func TestLevelUp_AllocatesThreePointsWithinCaps(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(4242)

	for iter := 0; iter < 500; iter++ {
		c := Character{Race: gamedata.Dwarf, MaxHP: 10, MaxSP: 10}
		for i := 0; i < gamedata.NumTraits; i++ {
			c.Traits[i] = 5
		}

		res, err := LevelUp(r, tb, &c)
		if err != nil {
			t.Fatalf("LevelUp: %v", err)
		}
		if res.Skipped {
			t.Fatal("屬性遠低於上限，不應跳過分配")
		}

		total := 0
		for i, g := range res.TraitGains {
			total += g
			m, _ := tb.RaceMax(gamedata.Dwarf, gamedata.Trait(i))
			if c.Traits[i] > m {
				t.Fatalf("屬性 %d = %d 超過種族上限 %d", i, c.Traits[i], m)
			}
		}
		if total != 3 {
			t.Fatalf("分配了 %d 點，預期 3 點", total)
		}
	}
}

// FromSave 與 ApplyTo 必須成對：讀進來再寫回去，讀得到的欄位都不能變。
//
// 兩支函式分開維護時最容易出的錯是「FromSave 多讀一個欄位、ApplyTo 忘了寫」，
// 那個欄位會在存檔時悄悄退回舊值 —— 玩家升級完存檔，回來發現等級沒了。
func TestCharacter_FromSaveApplyToRoundTrip(t *testing.T) {
	orig := scenario.Character{
		Name: "Wopple", RaceByte: 1, ClassByte: 6, Level: 3,
		Experience: 12345, MaxHP: 24, CurrentHP: 20,
		MaxSPBonus: 29, CurrentSP: 17,
		SpeedNatural: 11, StrengthNatural: 9, Intellect: 18,
		Endurance: 12, SkillNatural: 14,
	}
	orig.SkillFlags[gamedata.SkillPersuasion] = 1
	orig.SkillFlags[gamedata.SkillBerserking] = 1

	back := orig
	FromSave(orig).ApplyTo(&back)

	if back.Name != orig.Name || back.RaceByte != orig.RaceByte ||
		back.ClassByte != orig.ClassByte || back.Level != orig.Level ||
		back.Experience != orig.Experience {
		t.Errorf("基本欄位走樣：\n得到 %+v\n預期 %+v", back, orig)
	}
	if back.MaxHP != orig.MaxHP || back.CurrentHP != orig.CurrentHP ||
		back.MaxSPBonus != orig.MaxSPBonus || back.CurrentSP != orig.CurrentSP {
		t.Error("生命／法力走樣")
	}
	if back.SpeedNatural != orig.SpeedNatural ||
		back.StrengthNatural != orig.StrengthNatural ||
		back.Intellect != orig.Intellect || back.Endurance != orig.Endurance ||
		back.SkillNatural != orig.SkillNatural {
		t.Error("屬性走樣")
	}
	if back.SkillFlags != orig.SkillFlags {
		t.Errorf("技能旗標走樣：%v vs %v", back.SkillFlags, orig.SkillFlags)
	}
}

// 規則層不認識的欄位一個都不能動。
func TestCharacter_ApplyToLeavesUnknownFields(t *testing.T) {
	rec := scenario.Character{
		Name: "X", WeaponSlotIndex: 3, ArmorSlotIndex: 8,
		CombatStatus: scenario.StatusPoison, Unknown103: 0x5a,
		StrengthBonus: 99, SkillBonus: 98, SpeedBonus: 97, MaxSPNatural: 96,
	}
	before := rec
	FromSave(rec).ApplyTo(&rec)

	if rec.WeaponSlotIndex != before.WeaponSlotIndex ||
		rec.ArmorSlotIndex != before.ArmorSlotIndex ||
		rec.CombatStatus != before.CombatStatus ||
		rec.Unknown103 != before.Unknown103 {
		t.Error("裝備槽／戰鬥旗標／未知欄位被改寫了")
	}
	// 「含裝備加成」的欄位由裝備推導，規則層不該直接寫。
	if rec.StrengthBonus != before.StrengthBonus ||
		rec.SkillBonus != before.SkillBonus ||
		rec.SpeedBonus != before.SpeedBonus ||
		rec.MaxSPNatural != before.MaxSPNatural {
		t.Error("含加成欄位被改寫了")
	}
}

// 裝備要真的換算成戰鬥數值。
//
// 少帶裝備的話角色會空手、零護甲上場，戰鬥數字全部偏掉，
// 但畫面上看不出哪裡不對 —— 這是最容易被漏掉的一類缺陷。
func TestCharacter_EquipmentToCombatUnit(t *testing.T) {
	var c Character
	c.Name = "Kern"
	c.Level = 3
	c.CurrentHP, c.MaxHP = 20, 30
	c.Traits[gamedata.Speed] = 11
	c.Traits[gamedata.Strength] = 9

	// 第 0 格闊劍（ITEMS.DAT 索引 5），附魔 +2、特效 +3。
	c.Inventory[0] = scenario.InventorySlot{Type: 5, Enchant: 2, WeaponEffect: 3}
	// 第 1 格鎖子甲（索引 10 → 防護 3）。
	c.Inventory[1] = scenario.InventorySlot{Type: 10}
	c.EquippedWeapon = 0
	c.EquippedArmor = 1

	u := c.CombatUnit(PlayerSlotStart, 8, 1, West)

	if u.WeaponIndex != 6 {
		t.Errorf("武器骰索引 = %d，預期 ITEMS.DAT 索引 5 + 1 = 6", u.WeaponIndex)
	}
	if u.Armor != 3 {
		t.Errorf("護甲 = %d，預期鎖子甲的 3", u.Armor)
	}
	if u.EnchantBonus != 2 || u.WeaponEffect != 3 {
		t.Errorf("附魔／特效 = %d／%d，預期 2／3", u.EnchantBonus, u.WeaponEffect)
	}
	if !u.IsPlayer || u.Speed != 11 || u.HP != 20 || u.MaxHP != 30 {
		t.Errorf("基本欄位沒帶過來：%+v", u)
	}
}

// 沒裝備就是徒手、零護甲 —— 徒手在骰表的索引是 0，那一格是刻意留的。
func TestCharacter_UnarmedDefaults(t *testing.T) {
	var c Character
	for i := range c.Inventory {
		c.Inventory[i] = scenario.InventorySlot{Type: 0xff}
	}
	c.EquippedWeapon, c.EquippedArmor = 0, 1

	if got := c.WeaponDieIndex(); got != 0 {
		t.Errorf("空手的骰表索引 = %d，預期 0", got)
	}
	if got := c.ArmorRating(); got != 0 {
		t.Errorf("沒穿護甲的防護 = %d，預期 0", got)
	}
	if got := WeaponDamageDie(0); got != 2 {
		t.Errorf("徒手骰面 = %d，預期 2", got)
	}
}

// 五件護甲的防護值 1–5，對應 ITEMS.DAT 索引 8–12。
func TestCharacter_ArmorRatingRange(t *testing.T) {
	for typ := armorFirstIndex; typ <= armorLastIndex; typ++ {
		var c Character
		c.Inventory[0] = scenario.InventorySlot{Type: byte(typ)}
		c.EquippedArmor = 0
		want := typ - armorRatingBase
		if got := c.ArmorRating(); got != want {
			t.Errorf("索引 %d 的護甲防護 = %d，預期 %d", typ, got, want)
		}
	}
	// 武器不該被當成護甲。
	var c Character
	c.Inventory[0] = scenario.InventorySlot{Type: 5}
	c.EquippedArmor = 0
	if got := c.ArmorRating(); got != 0 {
		t.Errorf("把武器當護甲時防護 = %d，預期 0", got)
	}
}

// 裝備欄索引超出範圍不能 panic。
func TestCharacter_BadEquipIndex(t *testing.T) {
	var c Character
	for _, i := range []int{-1, InventorySlots, 999} {
		c.EquippedWeapon, c.EquippedArmor = i, i
		if !c.Weapon().Empty() || !c.Armor().Empty() {
			t.Errorf("索引 %d 應視為空槽", i)
		}
	}
}
