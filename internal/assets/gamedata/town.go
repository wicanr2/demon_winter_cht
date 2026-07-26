package gamedata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 城鎮資料。
//
// 執行期反組譯看到的是遠指標 `0x5534` 指向「當前城鎮資料表」，
// 價格公式從它的 `+0x1ed`（經濟係數）與 `+0x1f5`（船價基礎值）取值。
// **那個緩衝區就是載入的 `TOWN<n>.DAT` 檔案本身** —— 檔案剛好 512 bytes，
// 兩個位移都落在檔內。
//
// 這條對應是本專案自己確認的，兩個獨立證據：
//
//  1. `+0x1ed` 在 25 座城鎮的值落在 8–25，正好是「物價指數」該有的值域。
//  2. `+0x1f5` 只有 5 座城鎮非零（Janthrin、New Gleon、Dragontooth、
//     Asaht、Land's Edge），其餘 20 座為 0 —— 只有碼頭城鎮賣船。
//     攻略明寫「前往東北方很遠的**新格里昂**，買一艘船」，
//     New Gleon 正是那 5 座之一。
const (
	// NumTowns 是城鎮數。TOWN.TXT 前 25 個字串即 25 座城鎮名。
	NumTowns = 25

	// townRecordSize 是 TOWN<n>.DAT 的檔案大小。
	townRecordSize = 512

	// offTownEconomy／offTownShipBase 是城鎮表內的兩個已驗證欄位。
	offTownEconomy  = 0x1ed
	offTownShipBase = 0x1f5
)

// townSites 是 25 座城鎮在世界地圖上的 (X, Y)，索引 0 = 1 號城鎮。
//
// 來源：DEMON.INT 的 DS:0x2f86（檔位移 0x28a86），25 組 (X, Y) byte 對。
// 這是從機器碼硬讀出來的常數表，不是資料檔，所以照 sumMapIDs／sumMapSizes
// 的作法寫死在這裡（原版執行檔不入版控，執行期沒有它可讀）。
//
// 怎麼找到的：`town%d.dat` 這個格式字串在 DS:0x3055，全檔只有一處引用
// （0x1b605 的 `mov ax,0x3055`）。往前反組譯就是座標→城鎮編號的查表迴圈：
//
//	1b5d2  loop: inc word [bp-0x5c]        ; 索引 += 2（一組兩 byte）
//	1b5d5        inc word [bp-0x5c]
//	1b5d8        inc word [bp-0x58]        ; 城鎮編號++
//	1b5db  chk:  mov bx,[bp-0x5c]
//	1b5de        mov cl,[bx+0x2f86]        ; 表內 X
//	1b5e2        les bx,[0x4c76]           ; 隊伍狀態
//	1b5e6        mov al,es:[bx+0xa1]       ; 隊伍 X
//	1b5eb        cmp al,cl / jne loop
//	1b5f2        mov al,[bx+0x2f87]        ; 表內 Y，與 es:[bx+0xa2] 比
//	1b601        push [bp-0x58]            ; 命中 → 城鎮編號
//	1b605        mov ax,0x3055             ; "town%d.dat"
//
// 迴圈**沒有上界**，所以它只在「已經確定踩到城鎮」時才會被呼叫；判斷在
// 呼叫端（0x19192）：剛踩到的 tile 值存在 DS:0x52f4，等於 0x2e 或 0x64
// 才 call。編號 1-based 是因為 [bp-0x58] 從 1 起算。
//
// 表的邊界是資料本身給的：第 25 組 (1,3) 之後緊接著 `ec 2f f0 21`，
// 是另一張遠指標表，不是座標。
//
// 交叉驗證（獨立於反組譯）：把這 25 個座標拿去查地圖資料，24 個精確落在
// SUM.MAP 子地圖的城鎮 tile（0x2e，Asaht 是 0x64）上，剩下的海盜灣落在
// MAP1.MAP，25/25 全中。全世界的 0x2e／0x64 共 27 格，扣掉這 25 格只多
// 2 格（都在子地圖 54，語意未解）。這個比對同時反過來抓出了 SUM.MAP
// RLE 解壓的兩個 bug，見 world.decodeRLE 說明 §3。
var townSites = [NumTowns][2]int{
	{31, 45}, {46, 45}, {45, 34}, {37, 23}, {10, 39},
	{25, 24}, {57, 22}, {34, 11}, {57, 8}, {40, 55},
	{51, 40}, {33, 38}, {44, 32}, {39, 40}, {45, 15},
	{27, 26}, {14, 30}, {26, 9}, {10, 36}, {22, 31},
	{51, 44}, {21, 53}, {39, 44}, {41, 52}, {1, 3},
}

// TownTileValues 是「踩上去會進城」的 tile 值。原版在 0x19192 只比對這兩個
// 值就呼叫座標查表；0x64 目前只有 Asaht 一處用到。
var TownTileValues = [2]byte{0x2e, 0x64}

// IsTownTile 回報這個 tile 值踩上去是不是會進城。
func IsTownTile(v byte) bool {
	return v == TownTileValues[0] || v == TownTileValues[1]
}

// Town 是一座城鎮。
type Town struct {
	// Number 是 1–25，對應 TOWN<n>.DAT。
	Number int
	Name   string

	// X、Y 是這座城鎮在世界地圖上的座標（見 townSites）。
	X, Y int

	// Economy 是經濟係數 E，一切價格的基礎。
	Economy int
	// ShipBase 是買船價的基礎值，買船價 = ShipBase × 10。0 代表不賣船。
	ShipBase int

	// Facilities 是這座城鎮有哪些設施。
	Facilities TownFacilities

	// raw 是整份 512 bytes，其餘欄位語意未解，先留著供後續分析。
	raw []byte
}

// SellsShips 回報這座城鎮有沒有碼頭在賣船。
func (t Town) SellsShips() bool { return t.ShipBase > 0 }

// 城鎮設施旗標在 `TOWN*.DAT` 的固定絕對位移上，全部落在最後一筆
// 17-byte 記錄（record 29）的 payload 裡。原版的選單建構
// （`FUN_278d_????`，`278d:02bc` 起）逐個 `CMP … ,0 / JZ 跳過`，
// **市集永遠顯示、不查任何旗標**。詳見 `docs/re/02` §3.2。
const (
	offFacHealers = 0x1ee
	offFacInn     = 0x1ef
	offFacGuild   = 0x1f0
	offFacChurch  = 0x1f1 // 不是布林，是神殿所屬神祇的編號
	offFacCollege = 0x1f2 // 三個槽位，0xff 代表空
	offFacPub     = 0x1f6

	// 碼頭沒有自己的旗標 —— **`0x1f5` 一個位元組兼兩用**：
	// 非 0 就顯示碼頭選項（`278d:04a2`），值本身又是船價基礎
	// （`offTownShipBase`）。這與引擎其他地方「同一個值兩種用途」同源。

	// collegeSlots 是學院槽位數（278d:0586 的迴圈跑三輪）。
	collegeSlots = 3
	// collegeEmpty 是空槽（278d:059c 拿 0xff 比對）。
	collegeEmpty = 0xff
)

// TownFacilities 是一座城鎮有哪些設施。
type TownFacilities struct {
	// Market 恆為 true —— 原版無條件顯示，不查旗標。
	Market bool
	Healers, Inn, Guild, Docks, Pub bool

	// Church 是神殿所屬神祇的編號，0 代表沒有神殿。
	Church int
	// Colleges 是三個學院槽位裡實際有的那些（已濾掉 0xff）。
	Colleges []int
}

// Has 依設施編號回報有沒有那項設施。編號順序與 game.AllFacilities 一致
// （市集／治療所／旅店／公會／神殿／碼頭／酒館／學院），前七項與原版選單的
// 建構順序相同，學院是本作補的第八項。
func (f TownFacilities) Has(facility int) bool {
	switch facility {
	case 0:
		return f.Market
	case 1:
		return f.Healers
	case 2:
		return f.Inn
	case 3:
		return f.Guild
	case 4:
		return f.HasChurch()
	case 5:
		return f.Docks
	case 6:
		return f.Pub
	case 7:
		return len(f.Colleges) > 0
	}
	return false
}

// HasChurch 回報這座城鎮有沒有神殿。
func (f TownFacilities) HasChurch() bool { return f.Church != 0 }

// parseFacilities 解出設施旗標。
func parseFacilities(raw []byte) TownFacilities {
	f := TownFacilities{
		Market:  true,
		Healers: raw[offFacHealers] != 0,
		Inn:     raw[offFacInn] != 0,
		Guild:   raw[offFacGuild] != 0,
		Church:  int(raw[offFacChurch]),
		Docks:   raw[offTownShipBase] != 0,
		Pub:     raw[offFacPub] != 0,
	}
	for i := 0; i < collegeSlots; i++ {
		if v := raw[offFacCollege+i]; v != collegeEmpty {
			f.Colleges = append(f.Colleges, int(v))
		}
	}
	return f
}

// TownTable 是 25 座城鎮。
type TownTable struct {
	towns []Town
}

// LoadTownTable 從資料目錄讀入 TOWN.TXT 與 TOWN1..25.DAT。
//
// 缺任何一個檔就是錯誤 —— 城鎮少一座不會當掉，只會讓玩家某天走進去
// 發現物價全錯，那種缺陷比開不起來難查得多。
func LoadTownTable(dataDir string) (*TownTable, error) {
	names, err := loadTownNames(filepath.Join(dataDir, "TOWN.TXT"))
	if err != nil {
		return nil, err
	}

	t := &TownTable{}
	for i := 1; i <= NumTowns; i++ {
		path := filepath.Join(dataDir, fmt.Sprintf("TOWN%d.DAT", i))
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("gamedata: 讀取 %s 失敗: %w", path, err)
		}
		if len(raw) != townRecordSize {
			return nil, fmt.Errorf("gamedata: %s 長度 %d，預期 %d",
				path, len(raw), townRecordSize)
		}
		name := fmt.Sprintf("城鎮 %d", i)
		if i-1 < len(names) {
			name = names[i-1]
		}
		t.towns = append(t.towns, Town{
			Number:   i,
			Name:     name,
			Facilities: parseFacilities(raw),
			X:        townSites[i-1][0],
			Y:        townSites[i-1][1],
			Economy:  int(raw[offTownEconomy]),
			ShipBase: int(raw[offTownShipBase]),
			raw:      raw,
		})
	}
	return t, nil
}

// loadTownNames 讀 TOWN.TXT 的 NUL 分隔字串。
func loadTownNames(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gamedata: 讀取 %s 失敗: %w", path, err)
	}
	var out []string
	for _, part := range strings.Split(string(data), "\x00") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	if len(out) < NumTowns {
		return nil, fmt.Errorf("gamedata: %s 只解出 %d 個字串，至少要 %d 個城鎮名",
			path, len(out), NumTowns)
	}
	return out, nil
}

// Len 回傳城鎮數。
func (t *TownTable) Len() int { return len(t.towns) }

// All 回傳全部城鎮。
func (t *TownTable) All() []Town { return append([]Town(nil), t.towns...) }

// TownAt 以世界座標找城鎮。原版就是這樣做的：只比對 (X, Y)，不看在哪張
// 子地圖上 —— 25 組座標在整個世界裡不重複，所以夠用。
//
// 呼叫端要先確認踩到的 tile 是城鎮（IsTownTile）；原版的查表迴圈沒有上界，
// 座標不在表上就會一路讀過頭。這裡改成回 ok=false，不重現那個 bug。
func (t *TownTable) TownAt(x, y int) (Town, bool) {
	for _, town := range t.towns {
		if town.X == x && town.Y == y {
			return town, true
		}
	}
	return Town{}, false
}

// ByNumber 以 1–25 取城鎮。
func (t *TownTable) ByNumber(n int) (Town, error) {
	if n < 1 || n > len(t.towns) {
		return Town{}, fmt.Errorf("gamedata: 城鎮編號 %d 超出 1–%d", n, len(t.towns))
	}
	return t.towns[n-1], nil
}
