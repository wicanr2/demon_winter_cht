package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 地城陷阱：七種陷阱本體 ＋ 「已注意」的迴避（`docs/re/68`、`docs/re/91`）
//
// 進入點是 `nSS.DAT` 類別 3／6 的記錄（`25be:0263` ＝ `0x19a43`），
// **記錄的低 5 bit 就是九格分派表的 case 編號**（`ds:0x52f6`，`0x19ed7`）。
//
// 在這之前引擎只印一行「有陷阱！」——**七種陷阱的擲點早就全部讀出來了**
// （`docs/re/68`），而且與手冊「地底 → 陷阱」那一節逐項吻合。
// 這不是「還沒解」，是「解完沒接」。

// TrapCase 是九格陷阱分派表的 case 編號。
type TrapCase byte

const (
	// TrapUnknown 是 case 0（`Something you cannot`…），語意未讀。
	TrapUnknown TrapCase = 0
	// TrapPoisonNeedle 毒針：**唯一沒有命中判定**的陷阱。
	TrapPoisonNeedle TrapCase = 1
	// TrapPunjiPit 竹籤陷阱。執行檔字串誤拼成 `A Bungei pit!`；
	// 官方手冊的 `Punji pit` 描述與此 case 的 50%／1–6 規則逐項相同。
	TrapPunjiPit TrapCase = 2
	// TrapPoisonPit 毒坑。**數值與竹籤坑完全相同** —— 那一段沒有任何
	// `cmp ds:0x52f6`，只有宣告的字串不同（`docs/re/91` §3）。
	TrapPoisonPit TrapCase = 3
	// TrapSpears 長矛：飛鏢的加強版，多擲一顆 4 面骰。
	TrapSpears TrapCase = 4
	// TrapDarts 飛鏢。
	TrapDarts TrapCase = 5
	// TrapPool 水池。**與 tile 0x35 的治療水池是兩件事**（`docs/re/90` §3）。
	TrapPool TrapCase = 6
	// TrapAcidPool 酸池：水池的高危險版，每輪多 2 點。
	TrapAcidPool TrapCase = 7
	// TrapAlarm 警報：**整組陷阱裡唯一不碰 HP 的**。
	TrapAlarm TrapCase = 8
)

// 擲骰用的骰面數。全部照 `docs/re/68`／`91` 的運算元，**不要湊整數**。
const (
	// TrapNoticeAvoidDie 是「已注意」的迴避擲點（`0x19a92`，1 ＝ 躲過）。
	// 手冊「經過時可能不會觸發」的精確值就是這個 50%。
	TrapNoticeAvoidDie = 2

	// TrapNeedleDamageDie 毒針傷害 1–4（`0x19af1`）。護甲無效、必中。
	TrapNeedleDamageDie = 4

	// TrapPitSafeDie 坑的安全擲點（`0x19bb3`，1 ＝ `: safe`）。
	TrapPitSafeDie = 2
	// TrapPitDamageDie 坑的傷害 1–6（`0x19c06`）。
	TrapPitDamageDie = 6

	// TrapVolleyCountDie 齊射數量 ＝ Roll(5)+1 ＝ 2–6 支（`0x19c8b`）。
	TrapVolleyCountDie = 5
	// TrapVolleyHitDie 每一支各自判定（`0x19cc5`，1 ＝ 落空）。
	TrapVolleyHitDie = 2
	// TrapVolleyDamageDie 每一支的基礎傷害 1–3（`0x19d2e`）。
	TrapVolleyDamageDie = 3
	// TrapSpearBonusDie 只有長矛多擲這一顆（`0x19d52` 的 `cmp ds:0x52f6,4`），
	// 所以長矛 2–7、飛鏢 1–3 —— 與手冊逐字吻合。
	TrapSpearBonusDie = 4

	// TrapPoolEscapeDie 每一輪的脫困擲點（`0x19df3`，**2** ＝ `swims out.`）。
	// 1/3 ＝ 手冊的「每回合 33% 機率脫困」。
	TrapPoolEscapeDie = 3
	// TrapPoolEscapeRoll 是脫困的那一面。**不是 1** —— 原版比的是 2。
	TrapPoolEscapeRoll = 2
	// TrapPoolDamageDie 溺水傷害 ＝ Roll(3)−1 ＝ 0–2（`0x19e12`）。
	// **0 是合法結果**，原版照樣印一行 `Drowns. 0 damage`。
	TrapPoolDamageDie = 3
	// TrapAcidPoolBonus 酸池每輪多 2 點（`0x19e27` 兩個 `inc`）。
	TrapAcidPoolBonus = 2

	// TrapAlarmDie 警報把遭遇倒數設成 1–5 步（`0x19eac`）。
	TrapAlarmDie = 5
)

// TrapHit 是陷阱打出的一下。
type TrapHit struct {
	// Member 是被指到的隊員索引。陷阱選人是**純隨機**，與站位、職業、
	// 狀態都無關（`25be:153a` ＝ `Roll(人數)−1`，`docs/re/68` §2）。
	Member int
	// Missed 代表這一下落空（坑的 `: safe`、齊射的 `%smisses`、池的 `swims out.`）。
	Missed bool
	// Damage 是實際扣掉的點數。落空時是 0；**沒落空也可能是 0**（水池）。
	Damage int
	// Died 代表這一下讓他倒下。
	Died bool
}

// TrapResult 是一次觸發的完整結果。
type TrapResult struct {
	// Avoided 代表這格是「已注意」而且擲贏了迴避 —— 整個陷阱不發生。
	Avoided bool
	// Hits 依序記錄每一下。齊射會有 2–6 筆，水池會有一輪一筆。
	Hits []TrapHit
	// AlarmCountdown > 0 表示警報把遭遇倒數設成這個值（走幾步必定遇敵）。
	AlarmCountdown int
	// Wiped 代表全隊死亡。原版在每次套用 HP 之後都檢查一次
	// （`25be:145c` 回 0 → `25be:000c`）。
	Wiped bool
}

// SpringTrap 觸發一個陷阱。
//
// noticed 代表這格是類別 6（已被 `L` 標記過）。原版把這一段放在分派之前
// （`0x19a87`）：擲贏就整個跳過，擲輸還要多印兩行「即使再小心…」。
//
// **選到已經死掉的人時，這個陷阱就空放了** —— 原版在坑那一段明寫
// `if (char[+0x102] == 5) 跳到結束`（`0x19b67`）。這裡對所有陷阱一致處理：
// 死人不會再被扣血，而齊射的每一支各自選人，所以其餘幾支照樣有效。
func SpringTrap(r *rng.RNG, c TrapCase, noticed bool, party []Character) TrapResult {
	var res TrapResult
	if len(party) == 0 {
		return res
	}
	if noticed && r.Roll(TrapNoticeAvoidDie) == 1 {
		res.Avoided = true
		return res
	}

	switch c {
	case TrapPoisonNeedle:
		// 沒有命中判定 —— 這才是攻略特別點名它的原因（`docs/re/68` §1）。
		res.strike(r, party, pickTrapVictim(r, party), r.Roll(TrapNeedleDamageDie))

	case TrapPunjiPit, TrapPoisonPit:
		who := pickTrapVictim(r, party)
		if r.Roll(TrapPitSafeDie) == 1 {
			res.Hits = append(res.Hits, TrapHit{Member: who, Missed: true})
			break
		}
		res.strike(r, party, who, r.Roll(TrapPitDamageDie))

	case TrapSpears, TrapDarts:
		n := r.Roll(TrapVolleyCountDie) + 1
		for i := 0; i < n && !res.Wiped; i++ {
			if r.Roll(TrapVolleyHitDie) == 1 {
				res.Hits = append(res.Hits, TrapHit{Member: -1, Missed: true})
				continue
			}
			who := pickTrapVictim(r, party)
			dmg := r.Roll(TrapVolleyDamageDie)
			if c == TrapSpears {
				dmg += r.Roll(TrapSpearBonusDie)
			}
			res.strike(r, party, who, dmg)
		}

	case TrapPool, TrapAcidPool:
		// **選人只做一次，然後一輪一輪擲**（`0x19dba` 在迴圈外，
		// `0x19e96` 跳回 `0x19dd3`）—— 手冊的「每回合 33% 脫困」。
		who := pickTrapVictim(r, party)
		for !res.Wiped {
			if r.Roll(TrapPoolEscapeDie) == TrapPoolEscapeRoll {
				res.Hits = append(res.Hits, TrapHit{Member: who, Missed: true})
				break
			}
			dmg := r.Roll(TrapPoolDamageDie) - 1
			if c == TrapAcidPool {
				dmg += TrapAcidPoolBonus
			}
			if !res.strike(r, party, who, dmg) {
				break // 他倒下了，或本來就是死人 —— 迴圈的出口是 HP 歸零
			}
		}

	case TrapAlarm:
		// 不扣血。`You set off an alarm!` 的意思是「幾步之內必定遇敵」。
		res.AlarmCountdown = r.Roll(TrapAlarmDie)
	}
	return res
}

// strike 對一名隊員套用傷害，回傳他是否還活著。
//
// 對應原版的 `25be:145c`：新 HP > 0 就寫回；否則 HP 歸零、狀態設死亡，
// 然後**借每步 HP 那一支結算**（`ds:0x15ca = 0x80` → `222f:0619`，
// `docs/re/63` 的模式表第三列）。回傳值是「還有人活著嗎」。
//
// ⚠ **那個結算會讓全隊各掉 1 點 HP**，因為模式 `0x80` 就是符印區那條路。
// 讀起來很怪，但三處寫入只有這一種模式值，而且 `docs/re/63` 早就把
// `0x1acd1` 列進去了。**尚未用 DOSBox 複核**（`CONTEXT.md` §7 C 區）。
func (res *TrapResult) strike(r *rng.RNG, party []Character, who, dmg int) bool {
	if who < 0 || who >= len(party) {
		return true
	}
	c := &party[who]
	if c.Status == scenario.StatusDead {
		// 死人不再受傷，這一下等於空放（原版 `0x19b67`）。
		res.Hits = append(res.Hits, TrapHit{Member: who, Missed: true})
		return false
	}

	hit := TrapHit{Member: who, Damage: dmg}
	if c.CurrentHP > dmg {
		c.CurrentHP -= dmg
		res.Hits = append(res.Hits, hit)
		return true
	}

	// 新 HP <= 0 → 死亡分支。原版比的是 `<= 0`，所以扣到剛好 0 也算死。
	hit.Damage = c.CurrentHP
	hit.Died = true
	c.CurrentHP = 0
	c.Status = scenario.StatusDead
	res.Hits = append(res.Hits, hit)

	tick := StepHPTick(party, StepHPDrain)
	res.Wiped = tick.AllDead || PartyWiped(party)
	return false
}

// pickTrapVictim 隨機選一名隊員（`25be:153a` ＝ `Roll(人數)−1`）。
//
// **含死人**。原版不篩，篩掉會改變機率分布 —— 五人裡死了兩個時，
// 剩下三人各自被打中的機率是 1/5 不是 1/3。
func pickTrapVictim(r *rng.RNG, party []Character) int {
	return r.Roll(len(party)) - 1
}

// TrapCaseFor 把 `nSS.DAT` 記錄的低 5 bit 換成 case 編號。
//
// 值域外的一律當 case 0（未讀的那一格），不猜。
func TrapCaseFor(v byte) TrapCase {
	if v > byte(TrapAlarm) {
		return TrapUnknown
	}
	return TrapCase(v)
}

// --- `L` 查看陷阱（動作 `0x07` ＝ `222f:1882`，`docs/re/91` §1）---

// 察覺／解除陷阱的技能編號（`docs/re/21` §1 的 id 表）。
// 原版讀的是角色記錄 `+0xd2`／`+0xd3`，相對技能旗標基底 `0xc8` 就是這兩個。
const (
	SkillDetectTraps gamedata.SkillID = 10
	SkillDisarmTraps gamedata.SkillID = 11
)

// TrapSpot 是掃到的一格。
type TrapSpot struct {
	X, Y int
	// Index 是它在特殊格清單裡的位置。
	Index int
	// Disarmed 為真代表當場解除（整個 attr 清 0）；否則是標記「已注意」。
	Disarmed bool
}

// TrapScan 是一次 `L` 的結果。
type TrapScan struct {
	// Spots 依距離由近而遠。空的就是原版的 `No traps found.`。
	Spots []TrapSpot
	// HasDetect／HasDisarm 讓顯示層說得出為什麼（有沒有人會這兩個技能）。
	HasDetect, HasDisarm bool
}

// TrapDetectDie 是沒有察覺陷阱技能時的擲點（`0x1786a`，**擲中 4 才算找到**）。
// 手冊：「找到陷阱的機率是 25%…若隊伍裡有角色具備察覺陷阱技巧，機率會提高到 100%」。
const TrapDetectDie = 4

// trapScanFloor／trapScanGate 是掃描會穿過去的兩種 tile（`0x17829`）。
// 撞到別的就停 —— 對應手冊「陷阱只會設置在筆直的通道上」。
const (
	trapScanFloor = 0x00
	trapScanGate  = tileEventGateA
)

// LookForTraps 沿著面向掃描前方，找出並處理陷阱。
//
// 掃描距離是**光源強度**（原版 `ds:0x5c64`，來自存檔 `+0xa7`）——
// 手冊：「檢查你正前方、光源所能照到範圍內的所有格子」。
//
// 兩種處理照原版分開，**而且兩種都會寫進 `nSS.DAT`**（它是存檔，
// `docs/re/78`），所以離開地城再回來仍然有效：
//
//   - 有解除陷阱技能 → `Consume`（attr ← 0），這一格從此不是陷阱
//   - 沒有 → `Advance`（attr += 0x60，類別 3 → 6），之後觸發時有 50% 迴避
//
// 這兩支早就存在，是 `MarkVisited` 的兄弟；**不要為了「看得清楚」再抄一份**。
func LookForTraps(r *rng.RNG, party []Character, st *scenario.SpecialTiles,
	tiles TileSource, x, y int, f Facing, light int) TrapScan {

	var scan TrapScan
	for i := range party {
		c := &party[i]
		// 原版是 `+0x102 <= 1`：正常或中毒才算數，被束縛或死亡都不行。
		if c.Status > scenario.StatusPoison {
			continue
		}
		if c.HasSkill(SkillDetectTraps) {
			scan.HasDetect = true
		}
		if c.HasSkill(SkillDisarmTraps) {
			scan.HasDisarm = true
		}
	}
	if st == nil || tiles == nil {
		return scan
	}

	dx, dy := f.Delta()
	for step := 0; step <= light; step++ {
		x, y = x+dx, y+dy
		tile, err := tiles.TileAt(x, y)
		if err != nil {
			break
		}
		tile &= 0x7f
		if tile != trapScanFloor && tile != trapScanGate {
			break // 不是走道了
		}
		if tile != trapScanGate {
			continue // 是走道，但這一格不可能有記錄
		}
		hit := st.Lookup(byte(x), byte(y))
		if hit == nil {
			continue
		}
		if cls := hit.Tile.Class(); cls != scenario.SpecialClassTrap &&
			cls != scenario.SpecialClassTrapAlt {
			continue
		}
		if !scan.HasDetect && r.Roll(TrapDetectDie) != TrapDetectDie {
			continue // 25% 沒擲中 —— 手冊建議「特別懷疑的話掃兩次」
		}
		spot := TrapSpot{X: x, Y: y, Index: hit.Index, Disarmed: scan.HasDisarm}
		if scan.HasDisarm {
			st.Consume(hit.Index)
		} else {
			st.Advance(hit.Index)
		}
		scan.Spots = append(scan.Spots, spot)
	}
	return scan
}

// TrapNames 是九格的原文宣告字串，索引即 case 編號。
// 顯示層拿它當翻譯 key 的原文（與 `PartyDeathLines` 同一個做法）。
var TrapNames = [9]string{
	"Something you cannot",
	"Poison needle hits ",
	"A Bungei pit!",
	"Poison pit!",
	"Spear trap!",
	"Darts shoot from holes in the wall",
	"A pool!",
	"An acid pool!",
	"You set off an alarm!",
}
