package game

// 兩個看時辰的地點劇情：鐘（case 12）與旅人的床（case 13）
// （`docs/re/100` §2／§3）
//
// 這兩支都拿 `party+0x9f`（＝**時辰**）跟 `0x18` ＝ 24 比，方向相反：
//
//	case 12  hour >= 24  → 敲鐘有效（`0x1a46e` 的 JNC）
//	case 13  hour <= 24  → 才問要不要睡（`0x1a4bc` 的 JBE）
//
// 而床睡完把時辰**設成 25**。兩件事拼起來就是一條設計：
// 白天去找那位旅人睡一覺 → 時間跳到入夜 → 這時鐘才敲得響。
//
// > **這一對比較順帶佐證了「一天不只 23 小時」**（worklist C8 的疑問）。
// > 時辰上限如果是 23，`hour >= 24` 這條分支永遠到不了，
// > 那段「天使的哭聲」就是死碼。`hourWrap = 38`（時辰 1–37）才講得通。

// NightHour 是那兩個判斷共用的門檻（原版 `cmp …+0x9f, 0x18`）。
//
// **兩邊都含等於**：case 12 是 `>=`、case 13 是 `<=`，
// 所以剛好 24 時的話「既能睡也能敲」。照抄，不要挑一邊改成嚴格不等。
const NightHour = 24

// BellSleepHour 是睡醒之後的時辰（原版 `0x1a542` 的 `mov …+0x9f, 0x19`）。
//
// **直接設成 25，不是「加幾小時」，也不換日、不回血、不回法力。**
// 它純粹是把時鐘撥到入夜 —— 與紮營睡覺（`Clock.WakeAt`，會換日並回復）
// 是兩套不同的機制，不要合併。
const BellSleepHour = 25

// BellRung 是敲鐘的結果。
type BellRung int

const (
	// BellNothing：時辰還沒到，`Nothing happens.`
	BellNothing BellRung = iota
	// BellOpens：天使的哭聲，遠處那道門開了。
	BellOpens
)

// RingBell 回報這一次敲鐘的結果（原版 `0x1a46a`–`0x1a4b0`）。
func RingBell(hour int) BellRung {
	if hour >= NightHour {
		return BellOpens
	}
	return BellNothing
}

const (
	// BellDoorClosedTile／BellDoorOpenTile 是那道門的兩個 tile 值。
	//
	// `0x3a` 在可通行性表是 `0xff`（擋路），`0x39` 是 `0xfd`（可通行）——
	// 所以敲鐘是**把關著的門換成開著的門**。
	BellDoorClosedTile = 0x3a
	BellDoorOpenTile   = 0x39
)

// BellDoor 是那道門的座標（原版 `0x1a4aa` 寫線性索引 `0xed5`）。
//
// `0xed5 = 3797 = 59×64 + 21` → **(21,59)，地圖 1**。
// 換算與壓牆那邊同一套（緩衝區第 0 格就是 tile (0,0)，沒有檔頭那個 byte）。
//
// **鐘在 (26,8)，門在 (21,59) —— 隔了大半張地圖。** 玩家不會看到門打開，
// 只會聽到那句「天使的哭聲從天上傳來，然後一切又安靜了下來」。
var BellDoor = struct{ X, Y int }{21, 59}

// OpenBellDoor 把那道門換成開著的（回報有沒有真的動到）。
//
// 原版無條件寫入 —— 已經開了的話再敲一次也只是寫同一個值。
// 這裡照抄，只是把「本來不是那道關著的門」當成沒動到，方便軌跡與測試判讀。
func OpenBellDoor(m TileWriter) bool {
	if m == nil {
		return false
	}
	before, err := m.TileAt(BellDoor.X, BellDoor.Y)
	if err != nil {
		return false
	}
	if m.SetTileAt(BellDoor.X, BellDoor.Y, BellDoorOpenTile) != nil {
		return false
	}
	return before == BellDoorClosedTile
}
