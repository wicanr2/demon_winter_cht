package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 野外隨機遭遇。
//
// 原版是 `FUN_222f_2a5b`（Ghidra `222f:2a5b` = DEMON.INT 檔位移 `0x2694b`），
// 掛在事件動作分派表的 `case 0x16`。**觸發條件是倒數計時器
// `隊伍[+0x9c]` 歸零**（`0x16aee`），不是機率 —— 本文原本寫的
// 「觸發在 FUN_222f_0763（時間推進）」是把商隊那一擲當成了戰鬥，
// 見 `docs/re/51`。計時器初值、戰後、警報與海戰重設均已解並接入；
// 資料表的來歷與「地形就是可通行性值」見
// `internal/assets/gamedata/encounter.go` 與 `docs/re/24`。
//
// # 流程
//
//  1. 地形     = 可通行性表[腳下的 tile]
//  2. 群組     = 地形的 8 個槽位裡隨機一個（rnd(8)）
//     難度等級不落在該組的 [MinLevel, MaxLevel] 就整個重挑
//  3. 隻數     = 8 − rnd(7)                    → 1–7 隻（222f:2bf5）
//  4. 每一隻   = 該組 10 筆裡隨機一筆（rnd(10)），要通過等級檢查
//
// # 等級檢查（222f:2ca8–2cb8）
//
// 每一筆帶一個等級值 `L`，通過條件是 `難度 − 1 <= L <= 難度 + 1`。
// 也就是**遭遇會跟著隊伍變強**：等級 1 遇骷髏，等級 9 遇元素。
//
// `L >= 100` 的那種比較特別（222f:2c9e）：先減 100 再比，而且
//
//   - 只有**第一、二隻**能抽到（`已放數 <= 1`，222f:2c8e）
//   - 一旦抽中就**記住那一格**，這次遭遇剩下的每一隻都是同一種
//
// 所以沼澤那一組（十筆全是 `>= 100`）出來的一定是清一色的鬼火群或鬼墳族群，
// 不會混雜。這解釋了為什麼手冊要特別提醒那兩隻。
const (
	// EncounterChanceMask／EncounterChanceHit 是 `222f:081b–0823` 的
	// 1/64 擲點：`rnd_raw() & 0x3f == 0x34`。
	//
	// ⚠ **那一擲不是隨機戰鬥，是商隊**（`docs/re/51`）。它設的回傳碼是
	// `0x17`，而動作分派表的 `case 0x17` 是商隊遭遇；挑怪那一支
	// （`222f:2a5b`）掛在 `case 0x16`，由**倒數計時器 `隊伍[+0x9c]`**
	// 歸零觸發，不是機率。
	//
	// 常數本身沒錯，錯的是它接到哪裡。本專案現在拿它觸發商隊；
	// 隨機戰鬥走獨立的持久化倒數，不再使用這道 1/64。
	EncounterChanceMask = 0x3f
	EncounterChanceHit  = 0x34

	// encounterMaxMonsters 是隻數公式的被減數（222f:2c01 的 `MOV AX,8`）。
	encounterMaxMonsters = 8
	// encounterCountDie 是隻數公式擲的骰（222f:2bf5 的 `rnd(7)`）。
	encounterCountDie = 7

	// encounterGroupRetries 是挑不到合用群組時的重試上限。
	// 原版是無界重挑（0x2be7／0x2bf3 直接跳回去），難度值落在所有群組的
	// 範圍之外就會轉不出來；這裡設上限，回 nil 讓呼叫端當成「沒遇到」。
	encounterGroupRetries = 32
	// encounterEntryRetries 是單隻怪挑不到時的重試上限，理由同上。
	encounterEntryRetries = 32
)

// EncounterLevel 重現進地圖後的遭遇難度值。
//
// 基準來自存檔 +0xaf；踩 EXITS.DAT 出口時，第六欄會更新該值。
// 冬之魔降臨（劇情階段 2）後，原版再加 2，並把結果鉗在 10。
func EncounterLevel(base int, plotStage byte) int {
	if base < 1 {
		base = 1
	}
	if plotStage == 2 {
		base += 2
	}
	if base > 10 {
		base = 10
	}
	return base
}

// EncounterTriggered 回報這一步有沒有觸發隨機遭遇。
//
// raw 是 RNG 的原始輸出（原版直接對它做位元遮罩，不是 `rnd(64)`）。
// **只在戶外有效** —— 原版在子地圖編號 < 9（地城）與船上都不擲，
// 那兩個條件由呼叫端判斷。
func EncounterTriggered(raw int) bool {
	return raw&EncounterChanceMask == EncounterChanceHit
}

// EncounterCount 回傳這次遭遇出現幾隻：`8 − rnd(7)`，值域 1–7。
func EncounterCount(r RollSource) int {
	if r == nil {
		return 1
	}
	return encounterMaxMonsters - r.Roll(encounterCountDie)
}

// RollEncounter 依地形擲出一場遭遇，回傳 MONSTER.DAT 的索引。
//
// level 是遭遇難度（原版的 `ds:0x5c60`，1–10，跟著隊伍走）。
// 挑不出合用的群組或怪物時回 nil —— 呼叫端當成「這一步什麼都沒發生」。
func RollEncounter(r RollSource, tb *gamedata.Tables, terrain gamedata.Terrain,
	level int) []int {

	if r == nil || tb == nil {
		return nil
	}
	slots, err := tb.TerrainGroups(terrain)
	if err != nil {
		return nil
	}

	var group gamedata.EncounterGroup
	found := false
	for i := 0; i < encounterGroupRetries; i++ {
		g, err := tb.EncounterGroup(int(slots[r.Roll(gamedata.EncounterSlots)-1]))
		if err != nil || !g.Fits(level) {
			continue // 難度不合，回頭重挑（原版 0x2be7／0x2bf3）
		}
		group, found = g, true
		break
	}
	if !found {
		return nil
	}

	want := EncounterCount(r)
	out := make([]int, 0, want)
	// locked 是「抽中 >= 100 那種之後鎖定的格號」，−1 代表還沒鎖。
	locked := -1

	for len(out) < want {
		if locked >= 0 {
			out = append(out, group.Entries[locked].Monster)
			continue
		}
		e, idx, ok := rollEncounterEntry(r, group, level, len(out))
		if !ok {
			break // 這個難度挑不出東西，就出這麼多
		}
		if e.Gated() {
			locked = idx
		}
		out = append(out, e.Monster)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// rollEncounterEntry 從一組裡擲一筆合用的怪物。
//
// placed 是已經放進去幾隻 —— `>= 100` 的那種只有前兩隻抽得到（0x2c8e）。
func rollEncounterEntry(r RollSource, g gamedata.EncounterGroup, level, placed int) (
	gamedata.EncounterEntry, int, bool) {

	for i := 0; i < encounterEntryRetries; i++ {
		idx := r.Roll(gamedata.EncounterEntries) - 1
		e := g.Entries[idx]
		if e.Gated() && placed > 1 {
			continue // 已經放超過一隻就不能再開新的「清一色」群
		}
		if !e.Allowed(level) {
			continue
		}
		return e, idx, true
	}
	return gamedata.EncounterEntry{}, 0, false
}

// --- 隨機戰鬥的倒數計時器（存檔 `+0x9c`，見 `docs/re/51` §3）---

// 四個重設點。全部是 `rnd(n) + k` 的形狀。
const (
	// 新遊戲：15–19 步（`0x14933`，同一段還寫起始金幣 75）。
	countdownNewGameDie  = 5
	countdownNewGameBase = 14
	// 打完一場之後：28–77 步（`0xe1b3`，在戰鬥結算那一段）。
	// 兩份原版存檔的值（56 與 34）都落在這個範圍裡。
	countdownAfterBattleDie  = 50
	countdownAfterBattleBase = 27
	// 觸發警報（"You set off an alarm!"）：1–5 步（`0x19eb6`）。
	countdownAlarmDie = 5
)

// EncounterCountdownNewGame 回傳新遊戲的初值（15–19 步）。
func EncounterCountdownNewGame(r *rng.RNG) int {
	return r.Roll(countdownNewGameDie) + countdownNewGameBase
}

// EncounterCountdownAfterBattle 回傳打完一場之後的重設值（28–77 步）。
func EncounterCountdownAfterBattle(r *rng.RNG) int {
	return r.Roll(countdownAfterBattleDie) + countdownAfterBattleBase
}

// EncounterCountdownAlarm 回傳觸發警報之後的重設值（1–5 步）。
//
// `0x106ff` 的 151–200 後續已由 IDA 海戰收尾定案；由海戰呼叫端直接設定。
// 另一個常數 1 的事件脈絡與警報不同，不在這個 helper 合併。
func EncounterCountdownAlarm(r *rng.RNG) int { return r.Roll(countdownAlarmDie) }

// StepEncounterCountdown 走一步：把計時器減一，回報這一步要不要開打。
//
// **歸零才觸發**（`0x16aee` 的 `cmp 0 / jne 不觸發`）。減到 0 之後不再往下減 ——
// 原版是 byte 遞減，減過頭會繞回 255；本專案在觸發時就重設，不留那個窗口。
func StepEncounterCountdown(countdown int) (left int, fight bool) {
	if countdown > 0 {
		countdown--
	}
	return countdown, countdown == 0
}
