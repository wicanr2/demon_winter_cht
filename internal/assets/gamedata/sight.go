package gamedata

import "fmt"

// 戰場視線遮蔽。
//
// 這張表描述一個 9×9 的格子，中央那格（4,4）是視點。
// （**不是戰場** —— 戰場是 15×15，見下方的警告段落。）格子上如果站著會擋
// 視線的地形（樹、岩石之類，見 SightBlockerTiles），它後面的格子就看不到 ——
// 畫出來是空的。哪些格子被擋住不是算出來的，是查表：`FILES.DAT` 0x0a8
// 起的 176 bytes 就是那張表。
//
// # 表的結構
//
// 49 組，對應 9×9 網格**內部** 7×7（x=1..7、y=1..7）逐列掃過去的每一格。
// 每組列出「這一格會擋住哪些格子」，元素是 0–80 的網格索引（= y*9+x），
// 最高位標記該組的最後一個元素。49 組用掉 161 bytes，剩下 15 bytes 沒用到。
//
// # 怎麼確認的
//
// 讀取端在 `DEMON.INT` 檔位移 0x17455–0x174e9（表的遠指標在 DS:0x5504，
// 即資源 arena 第 6 段 = FILES.DAT 0x0a8，見 docs/re/22 §1）：
//
//	17455  mov [bp-0xc],0xa      ; 游標從 10 開始 = 網格 (1,1)
//	1745a  mov [bp-0xe],7        ; 內層跑 7 次
//	1746a  mov al,es:[bx+si]     ; 讀該格的地形 tile
//	1746d  and al,0x7f
//	17472  …                     ; tile 屬於 SightBlockerTiles 才成立
//	174a4  and ax,0x7f           ; 取表中元素
//	174b5  or  ax,0x80           ; 在網格上把該格的最高位打開 = 被遮蔽
//	174c4  inc [bp-0x18]         ; 表指標前進（不論成不成立都消耗一組）
//	174cf  jl  17491             ; 讀到最高位為 1 的元素才換下一組
//	174d1  dec [bp-0xe]
//	174dd  add [bp-0xc],2        ; 跳過邊界兩格
//	174e3  cmp [bp-0xc],0x49     ; 到 73 為止
//
// 游標序列 10–16、19–25、…、64–70 正好是 9×9 的內部 7×7，共 49 格 ——
// 與表切出來的組數精確相符（49 組、剛好用掉 161 bytes 而不是 160 或 162）。
//
// 遮蔽的效果在檔位移 0x1306d：掃過全部 81 格，最高位是 1 的一律寫成 0
// （空白）再交給算繪。所以是「看不到」，不是「不能走」。
//
// 幾何上也自洽：161 個遮蔽格**全部**都比投影它的那一格離中心更遠、
// 而且方向一致，沒有半個例外（`TestSightShadow_PointsAwayFromCentre`）。
// 這就是以中央為視點往外投影的陰影錐。
//
// # ⚠ 這張表的 9×9 **不是戰場**
//
// 這一段原本寫著「附帶結論：戰場確實是 9×9」。**那個推論是錯的。**
// 表是 9 寬沒問題，錯的是把它讀成「戰場只有 9 格寬」——
// 原版的戰場是 15×15（`docs/re/36`），9×9 是別的東西：
// 有一塊 `[0x514e]` 的 9×9 緩衝由 `0x172f4` 依光照內縮填入，
// 這張表大概是配它用的。**這張表目前沒有接到戰鬥上**，接回哪個畫面還沒查。
//
// 以下保留原本的推導 —— 表的形狀那部分仍然成立。
//
// `game.BattleGridWidth/Height` 原本註明「取 9×9 是為了與呈現層一致，
// 未經原版確認」。這張表把它釘死了：索引是 9 的倍數展開、上限 81、
// 內部掃描 7×7，三者只有在 9×9 時才自洽。
const (
	// offSightShadow 是遮蔽表在 FILES.DAT 裡的起點。
	offSightShadow = 0x0a8

	// sightShadowLen 是這一段的長度（資源 arena 第 6 段）。
	sightShadowLen = 176

	// SightGridSize 是戰場網格的邊長。
	SightGridSize = 9

	// sightCellCount 是網格總格數。
	sightCellCount = SightGridSize * SightGridSize

	// sightInteriorMin／Max 是會投影陰影的格子範圍（含）。
	// 最外圈不投影 —— 它後面已經沒有格子了。
	sightInteriorMin = 1
	sightInteriorMax = SightGridSize - 2

	sightGroupCount = (sightInteriorMax - sightInteriorMin + 1) *
		(sightInteriorMax - sightInteriorMin + 1) // 49
)

// SightBlockerTiles 是會擋住視線的地形 tile。
//
// 判斷式在 0x17472：`0x5d < tile < 0x62` 這一段，加上五個個別列舉的值。
// 這些 tile 的地形語意已經對上了（見 encounter.go 的 Terrain 與 `docs/re/24`），
// 不過「哪些會擋視線」是從機器碼直接讀出來的，不依賴那個對照。
var SightBlockerTiles = []byte{
	0x0d, 0x12, 0x13, 0x2a, 0x31,
	0x5e, 0x5f, 0x60, 0x61,
}

// BlocksSight 回報這個 tile 會不會擋住視線。tile 應已遮罩過 &0x7f。
func BlocksSight(tile byte) bool {
	for _, v := range SightBlockerTiles {
		if v == tile {
			return true
		}
	}
	return false
}

// SightShadow 是解好的遮蔽表。
type SightShadow struct {
	// shadow[i] 是「內部第 i 格」擋住的網格索引清單，
	// i 依 x=1..7、y=1..7 逐列排列（與原版掃描順序相同）。
	shadow [sightGroupCount][]int
}

// parseSightShadow 從 FILES.DAT 的原始 bytes 解出遮蔽表。
func parseSightShadow(data []byte) (*SightShadow, error) {
	if len(data) < offSightShadow+sightShadowLen {
		return nil, fmt.Errorf("gamedata: FILES.DAT 長度 %d 不足以容納視線遮蔽表", len(data))
	}
	seg := data[offSightShadow : offSightShadow+sightShadowLen]

	s := &SightShadow{}
	group, used := 0, 0
	var cur []int
	for i, b := range seg {
		if group >= sightGroupCount {
			used = i
			break
		}
		// 0xff 是「這一格什麼都遮不到」的哨兵：讀取端在 0x1749f 明確
		// 比對 0xff 並跳過標記。它同時也帶最高位，所以自己就是一組的結尾。
		if b != 0xff {
			v := int(b & 0x7f)
			if v >= sightCellCount {
				return nil, fmt.Errorf("gamedata: 視線遮蔽表第 %d 組出現網格索引 %d，超出 0–%d",
					group, v, sightCellCount-1)
			}
			cur = append(cur, v)
		}
		if b&0x80 != 0 {
			s.shadow[group] = cur
			cur, group = nil, group+1
			used = i + 1
		}
	}
	if group != sightGroupCount {
		return nil, fmt.Errorf("gamedata: 視線遮蔽表只解出 %d 組，預期 %d（9×9 網格內部 7×7）",
			group, sightGroupCount)
	}
	_ = used
	return s, nil
}

// sightIndex 把網格座標換成遮蔽表的組別索引，ok=false 代表這一格不投影陰影。
func sightIndex(x, y int) (int, bool) {
	if x < sightInteriorMin || x > sightInteriorMax ||
		y < sightInteriorMin || y > sightInteriorMax {
		return 0, false
	}
	span := sightInteriorMax - sightInteriorMin + 1
	return (y-sightInteriorMin)*span + (x - sightInteriorMin), true
}

// ShadowAt 回傳站在 (x, y) 的擋視線地形會遮住哪些格子，元素是網格索引
// （= y*9+x）。最外圈與界外一律回傳 nil —— 它們不投影陰影。
//
// 回傳的是內部切片，呼叫端不可修改。
func (s *SightShadow) ShadowAt(x, y int) []int {
	i, ok := sightIndex(x, y)
	if !ok {
		return nil
	}
	return s.shadow[i]
}

// HiddenCells 掃過整張 9×9 地形，回傳哪些格子被遮住看不到。
// （呼叫端目前沒有 —— 見檔頭的警告段落。）
//
// tiles 是 81 格的地形值（row-major，索引 = y*9+x）。長度不符回傳 error，
// 不 panic。
func (s *SightShadow) HiddenCells(tiles []byte) ([]bool, error) {
	if len(tiles) != sightCellCount {
		return nil, fmt.Errorf("gamedata: 戰場地形長度 %d，預期 %d",
			len(tiles), sightCellCount)
	}
	hidden := make([]bool, sightCellCount)
	for y := sightInteriorMin; y <= sightInteriorMax; y++ {
		for x := sightInteriorMin; x <= sightInteriorMax; x++ {
			if !BlocksSight(tiles[y*SightGridSize+x] & 0x7f) {
				continue
			}
			for _, c := range s.ShadowAt(x, y) {
				hidden[c] = true
			}
		}
	}
	return hidden, nil
}
