package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"

// AI 選法術。
//
// 原版是 `FUN_138d_065e`（Ghidra 138d:065e = DEMON.INT 檔位移 0x7b2e）。
// 怪物 AI 在法力 > 0 且 `rnd(10) > 4` 時走這一支（見 ai.go）。
//
// # 選法的兩層擲點
//
//	7b3a  rnd(5)                  ; 先擲一個符文系（1–5）
//	7b9f  mov al,[bx+0x1681]      ; hi = 範圍表[系+1]
//	7ba6  mov al,[bx+0x1680]      ; lo = 範圍表[系]
//	7bb2  rnd(hi − lo)            ; 再擲系內的第幾個
//	7bc5  add ax,cx / dec ax      ; 法術 id = lo + rnd(hi−lo) − 1
//	7bcc  call 0000:114f          ; 載入該法術的參數（school/effect/K/M）
//	7be5  mov ax,[bx+0x4ec2]      ; 施法者法力
//	7be9  cmp ax,[0x4e32]         ; 對比該法術的 M
//	7bed  jl 7b9c                 ; 不夠 → 回頭重挑
//
// `DS:0x1680` 起的位元組表用 `[bx]`／`[bx+1]` 兩個相鄰位移去讀，
// 是一張 **CSR 式的區間索引**：第 s 系的法術 id 落在 `[T[s], T[s+1])`。
// 這與噴吐那邊「同一張表用兩個基底讀」是同一種寫法。
//
// # 表的內容與佐證
//
// T = {0, 0, 4, 10, 14, 19, 24, 27, …}，所以：
//
//	第 1 系  id 0–3    （4 個）
//	第 2 系  id 4–9    （6 個）
//	第 3 系  id 10–13  （4 個）
//	第 4 系  id 14–18  （5 個）
//	第 5 系  id 19–23  （5 個）
//
// 五系連續涵蓋 id 0–23。下一段 `T[6]=24, T[7]=27` 是 id 24–26 ——
// 而**召喚是 0x18=24、幻術是 0x19=25**（既有結論），正好落在那一段，
// 且 AI 擲的是 `rnd(5)` 所以永遠抽不到它們。兩邊互相印證。
//
// # 會不會該系
//
// 0x7b5a–0x7b76 那一支會去查施法者的技能旗標（記錄內位移 `0xd8+r`，
// 落在 `skillFlags` `0xc8` 起的陣列，換算成技能 id 就是 **17–21**
// —— 正好是五個符文系，見 `docs/re/21`）。不會該系就不施法。
//
// 那一支的觸發條件是 `unit+0x20 == 2`，也就是**被附身、現在替怪物打的玩家
// 角色**（見 side.go）—— 它人還在玩家槽位、PC 記錄還在，所以要查它自己的
// 技能。一般怪物（陣營 1）不走這一支。
//
// 這裡仍把技能檢查做成可選參數，由呼叫端依陣營決定要不要傳。
//
// # 投入多少法力
//
// 挑定之後，AI **不是把法力全投下去**（0x7ceb–0x7d2e）：
//
//	7ceb  ax = 施法者法力
//	7cef  sub ax,[0x4e32]      ; 減掉該法術的 M → 餘裕
//	7cf6  rnd(100)
//	7d05  imul 餘裕
//	7d07  cx = 250
//	7d0d  call 2016:000a       ; 除法
//	7d12  add ax,[0x4e32]      ; 再加回 M
//	7d2e  sub [bx+0x4ec2],ax   ; 從法力扣掉
//
// 也就是 `投入 = M + rnd(100) × (法力 − M) / 250`。`rnd(100)` 是 1–100，
// 所以最多只會投入餘裕的 40%。見 AISpellInvestment。
//
// # 挑定之後打誰
//
// 0x7bef 之後有幾道以 `effect_type`（`ds:0x4e2e`）為條件的分支。
// `== 0xf` 是召喚／幻術那一支（0x7c07 依法術 id 24／25 把 effect_type
// 改寫成 4／2，對上「召喚成本 = PowerBase×4、幻術 = ×2」的既有結論）。
//
// 效果分派在 `0x7d35` 的 `jmp 8157`，**已解** —— 17 項跳表，
// 其中效果 3–7 依 K 的正負決定打自己人還是打玩家。見 aicast.go 與
// `docs/re/23`。
//
// 仍未解：`== 1` 且 `ds:0x518e == 1` 時重挑的那個狀態旗標。

// AISpellSchools 是 AI 會抽到的符文系數量。
const AISpellSchools = 5

// aiSchoolSkill 把符文系（1–5）換成技能 id（17–21）。
func aiSchoolSkill(school int) int { return 16 + school }

// aiSpellRange 是 DS:0x1680 的區間索引表：第 s 系的法術 id 落在
// [aiSpellRange[s], aiSpellRange[s+1])。索引 0 沒有用到。
var aiSpellRange = [8]int{0, 0, 4, 10, 14, 19, 24, 27}

// aiSpellRetries 是挑不到就重試的上限。
//
// 原版的重挑是無界迴圈（0x7bed 的 `jl` 直接跳回去）—— 施法者法力低於
// 全部法術的 M 時它會轉不出來。這裡設一個上限，回 ok=false 讓呼叫端
// 改做別的事。
const aiSpellRetries = 32

// AISpellChoice 依原版的兩層擲點替 AI 挑一個法術。
//
// knowsSkill 回報施法者會不會某個技能 id；傳 nil 代表不檢查
// （原版只對「被附身的玩家角色」檢查，見上）。
//
// 回傳法術 id 與是否挑到。法力不足以支付該法術的 M 就重挑，
// 重試上限內都挑不到就回 ok=false。
func AISpellChoice(r RollSource, tb *gamedata.Tables, currentSP int,
	knowsSkill func(skillID int) bool) (int, bool) {

	if r == nil || tb == nil {
		return 0, false
	}
	for i := 0; i < aiSpellRetries; i++ {
		school := r.Roll(AISpellSchools) // 1–5
		if knowsSkill != nil && !knowsSkill(aiSchoolSkill(school)) {
			continue
		}
		lo, hi := aiSpellRange[school], aiSpellRange[school+1]
		if hi <= lo {
			continue
		}
		id := lo + r.Roll(hi-lo) - 1
		sp, err := tb.Spell(id)
		if err != nil || sp.Empty() {
			continue
		}
		if currentSP < sp.M {
			continue // 法力不夠，回頭重挑（原版 0x7bed）
		}
		return id, true
	}
	return 0, false
}

// AISpellInvestment 回傳 AI 這次要投入多少法力。
//
// 公式照原版（0x7ceb–0x7d16）：`M + rnd(100) × (法力 − M) / 250`。
// `rnd(100)` 回 1–100，所以投入量落在 `M` 到 `M + 餘裕×0.4` 之間 ——
// **AI 不會把法力一次燒光**，這與玩家「預設全投」的介面不同。
//
// 法力不足 M 時回 currentSP（呼叫端本來就該先擋掉這種情況）。
func AISpellInvestment(r RollSource, currentSP, m int) int {
	if currentSP <= m {
		return currentSP
	}
	spare := currentSP - m
	return m + r.Roll(100)*spare/250
}

// RollSource 是 AISpellChoice 需要的擲點介面。
//
// 收介面而不是 *rng.RNG，是為了讓測試能餵固定序列，
// 把「挑法術」的邏輯與亂數本身分開驗。
type RollSource interface {
	Roll(n int) int
}
