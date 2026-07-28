package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 戰鬥外施法（紮營選單的 Cast，以及之後的 Worship）。
//
// 原版走的是與戰鬥同一支 `FUN_1000_11e5` —— 效果記錄、力度、判定全部共用，
// 差別只在「不在戰場上」。本專案原本的施法完全綁在 `Battle` 上
// （`applySpell` 要 `battle.Kill`／`battle.UnitAt`），所以紮營選單的
// Worship／Cast 一直卡著。這個檔案把不需要戰場的那一半拆出來。
//
// # 哪些效果離得開戰場
//
// 拆的標準不是「哪些好做」，是**哪些在戰鬥外有意義**：
//
//   - 生命、法力、解除束縛、枯萎 → 作用在角色的持久欄位，離得開
//   - 技巧／力量／速度／護甲增減 → 改的是**戰鬥單位**的暫時數值，
//     一離開戰場就沒了。在營地放等於什麼都沒做 —— 擋下來，不要假裝有效
//   - 範圍傷害、即死 → 要戰場網格與勝負結算，離不開

// CampCastResult 是一次戰鬥外施法的結果。
type CampCastResult struct {
	// OK 為 true 代表法術真的放出去了（法力已扣）。
	OK bool
	// Reason 是沒放成的原因。
	Reason string
	// Delta 是生命或法力的變化量（其餘效果為 0）。
	Delta int
	// Released 為 true 代表解開了束縛。
	Released bool
	// Withered 為 true 代表枯萎生效。
	Withered bool
	// Died 為 true 代表目標被自己人打死了（生命降到 0）。
	Died bool
}

// campCastStatusLimit 是能施法的狀態上限，與其他營地行動一致。
const campCastStatusLimit = 2

// CampCastable 回報這個效果在戰鬥外放不放得出來。
func CampCastable(effect int) (bool, string) {
	switch effect {
	case EffectHP, EffectSPMod, EffectBindRelease, EffectWither:
		return true, ""
	case EffectSkillMod, EffectStrengthMod, EffectSpeedMod, EffectArmorMod:
		return false, "這種增減只在戰鬥中有意義"
	case EffectAOE, EffectInstantDeath, EffectBindApply:
		return false, "這個法術要在戰鬥中才放得出來"
	}
	return false, "這個法術還不能在營地施放"
}

// CampCast 在營地施放一個法術。
//
// sp 是投入的法力，與戰鬥中同一個意思（力度）。**法力先扣再判定** ——
// 與戰鬥中一致，放空了也要付錢。
func CampCast(r *rng.RNG, caster, target *Character, s gamedata.Spell, sp int) CampCastResult {
	switch {
	case caster == nil || target == nil:
		return CampCastResult{Reason: "沒有這個人"}
	case int(caster.Status) >= campCastStatusLimit:
		return CampCastResult{Reason: "現在沒辦法施法"}
	case sp <= 0:
		return CampCastResult{Reason: "要投入法力才放得出來"}
	case caster.CurrentSP < sp:
		return CampCastResult{Reason: "法力不夠"}
	}
	if ok, why := CampCastable(s.Effect); !ok {
		return CampCastResult{Reason: why}
	}

	caster.CurrentSP -= sp

	// 借一個戰鬥單位當計算載體 —— 效果函式都吃 *Unit，
	// 而它們動到的欄位（HP／SP／狀態）在角色身上都有對應。
	u := target.CombatUnit(0, BattleCentreX, BattleCentreY, North)
	u.Status = UnitStatus(target.Status)

	res := CampCastResult{OK: true}
	switch s.Effect {
	case EffectBindRelease:
		res.Released = CastBindRelease(sp, s, u)
	case EffectWither:
		res.Withered = CastWither(r, sp, s, u)
	default:
		res.Delta, _ = CastMagnitudeEffect(r, sp, s, u)
	}

	writeBackFromUnit(target, u)
	res.Died = target.CurrentHP <= 0
	return res
}

// writeBackFromUnit 把戰鬥單位上會變動的持久欄位寫回角色。
//
// **只寫回持久的那幾項。** 速度／力量／技巧／護甲在單位上是戰鬥期間的
// 暫時值，寫回角色會把臨時加成變成永久 —— 那正是 `CampCastable` 擋掉
// 那四種效果的理由。
func writeBackFromUnit(c *Character, u *Unit) {
	c.CurrentHP = u.HP
	c.CurrentSP = u.CurrentSP
	if u.HP <= 0 {
		c.CurrentHP = 0
		c.Status = scenario.StatusDead
	} else {
		c.Status = scenario.CombatStatus(u.Status)
	}
}

// CampCastCandidates 列出這名角色在營地放得出來的法術索引。
//
// 過濾的是**效果類型**，不是技能 —— 本專案的施法選單目前不看技能
// （戰鬥中也一樣），那是另一個待補的坑。
func CampCastCandidates(t *gamedata.Tables) []int {
	if t == nil {
		return nil
	}
	var out []int
	for i := 0; i < t.NumSpells(); i++ {
		sp, err := t.Spell(i)
		if err != nil || sp.Empty() {
			continue
		}
		if ok, _ := CampCastable(sp.Effect); ok {
			out = append(out, i)
		}
	}
	return out
}

// WriteBackParty 把一場戰鬥結束後的持久狀態寫回隊伍。
//
// **在這之前，戰鬥完全沒有後果。** 引擎把角色複製成 `Unit` 上場，
// 傷害只寫在 `Unit` 上，而沒有任何地方寫回去 —— 打完一場慘勝回到地圖，
// 全隊照樣滿血；打輸了每個人也還是滿血，所以「全隊死亡」永遠不會成立。
//
// 症狀不會報錯，而且**戰鬥畫面上的血量是對的**（那是 `Unit` 的），
// 所以單看戰鬥截圖看不出來。要看出來得在戰鬥前後各看一次隊伍名冊。
//
// 對應到原版：角色記錄就是戰鬥用的那份資料（`docs/re/20`），
// 原版沒有「複製一份上場」這回事，所以它沒有這個問題也不需要這支函式 ——
// **這是移植架構自己引入的缺口**。
//
// 只寫回持久欄位，理由與 `writeBackFromUnit` 相同：
// 速度／力量／技巧／護甲在單位上是戰鬥期間的暫時值。
//
// 配對用**槽位**不用名字：`u2c` 那種按名字找的做法遇到同名角色會配錯，
// 而遊戲允許同名（建角不查重複）。
//
// 判別式只看槽位範圍，**不看 `Side`**。三段槽位是不相交的
// （怪物 0–6、隊伍 7–11、召喚 12–14，見 `battle.go`），所以槽位就夠了；
// 而且被魅惑的隊員 `Side` 會變成 `SideCharmedPlayer`，
// 用 `Side == SidePlayer` 篩會把他的傷害漏掉。
func WriteBackParty(members []Character, units []*Unit) {
	for _, u := range units {
		if u == nil || u.Slot < PlayerSlotStart || u.Slot >= PlayerSlotEnd {
			continue
		}
		i := u.Slot - PlayerSlotStart
		if i < 0 || i >= len(members) {
			continue
		}
		writeBackFromUnit(&members[i], u)
	}
}
