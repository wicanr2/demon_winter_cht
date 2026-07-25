package game

// 戰鬥單位的陣營欄位：單位記錄 `+0x20`（絕對位址 `DS:0x4ed4`）。
//
// 這個欄位曾經被判定為「語意未解」並公開收回過一次（見本檔末的說明）。
// 這一輪從六個互相獨立的位置把它釘死了。
//
// # 值域
//
// 全檔只有三個地方寫入立即值，另外兩個地方寫入計算值：
//
//	17c5:0962  MOV [BX+0x4ed4],0x1    ; 戰鬥單位建表：怪物側
//	17c5:0a66  MOV [BX+0x4ed4],0xb    ; 同一函式：玩家側（緊接著就去讀 PC 記錄）
//	138d:0330  MOV [BX+0x4ed4],0xb
//	138d:0dba  MOV [BX+0x4ed4],AX     ; 魅惑：把舊值對翻（見下）
//
// `17c5:05f8` 那個函式（2311 bytes）同時建立兩邊的單位，一邊寫 1、一邊寫
// 11，而寫 11 的那一支下一道指令就是 `LES BX,[0x4c7e]` 去讀角色記錄
// —— **11 是有 PC 記錄的那一邊**。
//
// # 門檻：>= 10 就是玩家側
//
// 全檔 40 幾處讀取，判別式只有兩種：跟 10 比大小，或直接跟 2／11 比對。
// 決定性的證據在 AI 施法的效果分派（`138d:0a4d`，效果類型 3–7）：
//
//	0a4d  CMP [0x4e30],0x0 / JLE 0x0a8c   ; K <= 0 → 走「打敵人」那一支
//	      ; K > 0（增益）：
//	0a83  CMP [BX+0x4ed4],0xa / JG  retry ; 目標必須 <= 10
//	      ; K <= 0（傷害／減損）：
//	0ab4  CMP [BX+0x4ed4],0xa / JL  retry ; 目標必須 >= 10
//
// 施法者是怪物。**增益挑 <= 10、傷害挑 >= 10** —— 所以 >= 10 是玩家側。
// 邊界值 10 兩邊都收，是原版自己的不一致（`JG` 對 `JL`）；實際值域裡
// 沒有 10，所以不會出事。照抄，不補對稱。
//
// # 魅惑會把陣營對翻
//
// `138d:0cb6` 那個函式（抗性 = `2×unit+0x06 + 法力`，`rnd(100)` 過關才生效）
// 在 `0x0d6a`–`0x0d88` 用一串比對把舊值換成新值，`0x0dba` 寫回：
//
//	1 → 12    11 → 2
//	2 → 11    12 → 1
//	4 → 14    14 → 4
//
// 是個完美的對合（involution），六個值剛好三對。低的三個（1、2、4）在
// 怪物側，高的三個（11、12、14）在玩家側。
//
// # 這就解釋了先前對不起來的那一處
//
// AI 施法一開頭有一道 `unit+0x20 == 2` 的閘（`138d:068a`），成立時去查
// 施法者的**符文系技能旗標**：
//
//	0691  MOV AX,0x104 / IMUL [BP+0x6]   ; 0x104 × 施法者槽位
//	0697  ADD AX,[BP-0xa]                ;   + 符文系（1–5）
//	069c  LES BX,[0x4c7e]                ; 角色記錄陣列
//	06a0  CMP ES:[BX+SI+0xf9bc],0x0      ; 旗標為 0 → 不施法
//
// `0xf9bc` 當有號 16-bit 看就是 `−0x644`，所以位移是 `0x104×槽位 − 0x644`。
// 槽位 7（隊伍第 0 人）→ `0x71c − 0x644 = 0xd8`，加上符文系 1–5 就是
// `0xd9`–`0xdd`；而角色記錄的技能旗標在 `0xc8`，技能 id 17–21 正好落在
// `0xd9`–`0xdd`（見 `docs/re/21`）—— 五個符文系，分毫不差。
//
// 也就是說 `== 2` 那一支只對**槽位 7–14** 成立。過去這被當成「`2` 是玩家側」
// 的反證，其實正好相反：**2 = 被魅惑、現在替怪物打的玩家角色**。
// 它人還在玩家槽位、PC 記錄還在，所以 AI 幫它施法前要查它自己的技能。
//
// 六處證據到這裡全部相容，沒有需要另外解釋的例外。
// 值域與配對表在 `docs/re/20` §9.5 就已經整理過（怪物 1、玩家 11、幻化 13、
// 召喚 14，配對 `1↔12`、`2↔11`、`4↔14`）。本輪從 AI 那一側重新推了一次，
// 結論一致 —— 過去 `ai.go` 掛著「語意未定案」的註記是**陳舊標記**，
// 不是真的未解。
const (
	// SideMonster 是一般怪物。建表時直接寫入（17c5:0962）。
	SideMonster = 1
	// SideCharmedPlayer 是被附身、改替怪物打的玩家角色。
	// AI 施法前查符文系技能旗標的那一支就是認這個值。
	SideCharmedPlayer = 2
	// SideEnemyIllusion 是怪物側的幻化生物。
	// **沒有配對值**，所以幻化生物不能被附身。
	SideEnemyIllusion = 3
	// SideEnemySummon 是被敵方奪取（或由怪物召喚）的召喚生物。
	SideEnemySummon = 4

	// SidePlayer 是一般玩家角色。建表時直接寫入（17c5:0a66）。
	SidePlayer = 11
	// SideCharmedMonster 是被附身、改替玩家打的怪物。
	SideCharmedMonster = 12
	// SideIllusion 是玩家側的幻化生物。同樣沒有配對值。
	SideIllusion = 13
	// SideSummon 是玩家側的召喚生物。召喚物算玩家陣營，會被友方 AOE 波及。
	SideSummon = 14
)

// sidePlayerThreshold 是「玩家側」的下界。原版在 40 幾處拿 10 去比。
const sidePlayerThreshold = 10

// sideFlip 是魅惑的對翻表（原版 138d:0d6a–0d88）。
// 幻化生物（3／13）**不在表上** —— 原版的比對鏈沒有這兩個值，走 default
// 什麼都不做。也就是幻影沒有心智可奪。
var sideFlip = map[int]int{
	SideMonster:       SideCharmedMonster,
	SideCharmedPlayer: SidePlayer,
	SideEnemySummon:   SideSummon,

	SidePlayer:         SideCharmedPlayer,
	SideCharmedMonster: SideMonster,
	SideSummon:         SideEnemySummon,
}

// SideIsPlayer 回報這個陣營值是不是玩家側。
func SideIsPlayer(side int) bool { return side >= sidePlayerThreshold }

// SideFlip 回傳被魅惑之後的陣營值。表上沒有的值原樣回傳、ok 為 false
// —— 原版遇到這種情況是什麼都不做（0x0d88 的 default 分支）。
func SideFlip(side int) (int, bool) {
	v, ok := sideFlip[side]
	if !ok {
		return side, false
	}
	return v, true
}

// defaultSide 是建表時的預設陣營值，對應 17c5:05f8 寫的兩個立即值。
func defaultSide(isPlayer bool) int {
	if isPlayer {
		return SidePlayer
	}
	return SideMonster
}

// EffectPossession 是附身術（POSSESSION）的效果類型。
//
// 它就是上面那張對翻表的唯一使用者：`138d:0c58` 的 `CALLF 0x1000:4586`
// 換算過來正是 `138d:0cb6`，也就是那個抗性判定 + 對翻的函式。
// 效果 0x10 的 AI 分支（`138d:0bef`）挑一次目標就算，不重挑。
const EffectPossession = 16

// possessionScale 是附身術成功率公式裡的常數 300（`138d:0cbf` 的 `0x12c`）。
const possessionScale = 300

// PossessionChance 回傳附身術的成功率（百分比，未鉗制）。
//
//	138d:0cbc  ax = 投入量
//	138d:0cbf  bx = 0x12c  / IMUL BX        ; 32-bit 乘法
//	138d:0cdd  ax = 目標 HP（unit+0x06）× 2
//	138d:0ce3  ax += 目標法力（unit+0x0e）
//	138d:0cfe  CALLF 0x3000:016a            ; 32-bit 除法
//
// 也就是 `投入 × 300 / (2×目標HP + 目標法力)`。目標愈健康、法力愈滿愈難附身。
//
// 分母為 0 時原版會除以零當掉；這裡回 0（等於必定失敗）。
func PossessionChance(invested, targetHP, targetSP int) int {
	d := 2*targetHP + targetSP
	if d <= 0 {
		return 0
	}
	return invested * possessionScale / d
}

// Possess 判定附身術是否成功，成功就把目標的陣營對翻。
//
// 成敗是單次擲骰：`rnd(100) <= 成功率`（`138d:0d20`–`0d2d`）。
// 失敗時原版印一句訊息就結束，目標不受影響。
func Possess(r RollSource, target *Unit, invested int) bool {
	if r == nil || target == nil {
		return false
	}
	if r.Roll(100) > PossessionChance(invested, target.HP, target.CurrentSP) {
		return false
	}
	side, ok := SideFlip(target.Side)
	if !ok {
		return false // 表上沒有的陣營值：原版的 default 什麼都不做
	}
	target.Side = side
	return true
}

// OnPlayerSide 回報這個單位這一刻替哪一邊打。
//
// 與 IsPlayer 不同：IsPlayer 說的是「有沒有 PC 記錄」（＝在不在槽位 7–14），
// 魅惑不會改變它；Side 說的是「現在替誰打」，魅惑會對翻。原版就是拿兩個
// 不同的東西表達這兩件事 —— 槽位決定記錄，`+0x20` 決定陣營。
func (u *Unit) OnPlayerSide() bool {
	if u == nil {
		return false
	}
	return SideIsPlayer(u.Side)
}
