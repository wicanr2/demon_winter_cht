package gamedata

import "fmt"

// 隨機遭遇：地形 → 遭遇群組 → 怪物。
//
// 這一段解掉了長期掛著的「手冊七大地形對到哪些 tile 值」。答案有點出人意料：
// **引擎沒有另一張地形表 —— 地形就是可通行性表的值。** 同一張
// `FILES.DAT 0x040` 被讀兩次：移動時當「能不能走／要不要多算一步」，
// 遭遇時當「這是哪一種地形」。這與本專案在別處看到的
// 「同一張表用兩個基底讀」是同一種 1980 年代省空間手法。
//
// # 原版的三段查表（`FUN_222f_2a5b`，檔位移 `0x2694b`）
//
//	222f:2a62  SI = Y×64 + X
//	222f:2a74  AL = 地圖[SI]                    ; tile 值
//	222f:2a7f  AL = [0x5500][tile]              ; ★ 可通行性表 → 地形類別
//	222f:2a82  [BP-1] = AL
//	...
//	222f:2b7b  rnd(8)
//	222f:2b8a  AX = 地形 × 8 + rnd(8)
//	222f:2b97  AL = [0x54fc][AX − 1]            ; ★ 地形的 8 個遭遇群組，隨機挑一個
//	...
//	222f:2bc5  SI = 群組 × 22
//	222f:2bda  AL = [0x550c][SI]                ; 群組最低等級
//	222f:2be9  AL = [0x550c][SI+1]              ; 群組最高等級
//	222f:2c50  AL = [0x550c][SI + 2 + 2×序號]   ; 怪物索引
//	222f:2c85  AL = [0x550c][SI + 3 + 2×序號]   ; 等級門檻
//
// `[0x54fc]` 與 `[0x550c]` 是資源區指標（見 `docs/re/22` §1），分別是
// `FILES.DAT 0x000`（64 bytes）與 `0x28E`（404 bytes）—— 那兩段原本都標「未知」。
// **64 = 8 地形 × 8 群組**，**404 ⊇ 18 群組 × 22 bytes**，兩個尺寸各自把
// 結構釘死。
//
// # 觸發條件
//
// `FUN_222f_0763`（時間推進）在每次行動後擲一次：
//
//	222f:080c  CMP [BX+0xa3],0x9 / JC 跳過      ; 子地圖編號 < 9 → 地城，不擲
//	222f:0814  CMP [0x52d4],0x0  / JNZ 跳過     ; 船上不擲
//	222f:081b  rnd_raw() & 0x3f == 0x34         ; 1/64
//
// 也就是**戶外每走一步有 1/64 機率遭遇**。
const (
	// offEncounterTerrain 是「地形 → 8 個遭遇群組」表（8×8 bytes）。
	offEncounterTerrain = 0x000
	// offEncounterGroups 是遭遇群組表。
	offEncounterGroups = 0x28E

	// NumTerrains 是可通行性值 0–7 這八個「地形類別」。
	NumTerrains = 8
	// EncounterSlots 是每個地形掛幾個遭遇群組。
	EncounterSlots = 8
	// NumEncounterGroups 是遭遇群組數。404 bytes 放得下 18 組（396），
	// 尾端 8 bytes 是別的東西，地形表也只引用到 17。
	NumEncounterGroups = 18
	// encounterGroupSize 是一組的位元組數：2 bytes 表頭 + 10 筆 × 2 bytes。
	encounterGroupSize = 22
	// EncounterEntries 是一組裡有幾筆怪物。
	EncounterEntries = 10

	// encounterLevelGated 是「清一色群」的分界。等級值 >= 100 的那種
	// 先減 100 再比等級，而且會把整場遭遇鎖定成同一隻（222f:2c9e）。
	encounterLevelGated = 100
)

// Terrain 是地形類別，值就是可通行性表的內容（0–7）。
//
// 手冊列了七大地形，這裡有八個類別，而且**對不成一對一**：
//
//	類別 0  森林 Forest    12 個 tile，圖是樹
//	類別 1  平原 Plains     9 個 tile，世界地圖上最多（15721 格）
//	類別 2  沼澤 Swamp      2 個 tile
//	類別 3  山丘 Hills      2 個 tile（0x0e／0x2b），移動多算一步
//	類別 4  地城地板        3 個 tile，MAP1／MAP5 的主要地板
//	類別 5  凍土 Tundra     1 個 tile（0x5a），黑底白點
//	類別 6  地城地板（另一種）1 個 tile（0x56），MAP3 的主要地板
//	類別 7  沙漠 Desert     5 個 tile
//
// **手冊的「Kudzu 樹叢」沒有自己的類別。** 它在手冊裡的怪物描述
// （類人族、探險者、昆蟲、龍）剛好是平原那一組的子集，推測是併進平原。
//
// 判定依據見各常數的說明；只有山丘是直接驗證（移動成本），
// 其餘是「怪物組成 + 圖塊外觀 + 世界地圖佔比」三路佐證的**推定**。
type Terrain byte

const (
	// TerrainForest 森林。12 個 tile 全是樹，遭遇組含棕熊／灰熊／森林狼／
	// 潛行者（Stalker）—— 手冊：「棲息熊、野狼、人類。不死怪物與昆蟲、
	// 毒蛇及神秘潛行怪物一起出現」，逐項對上。
	TerrainForest Terrain = 0
	// TerrainPlains 平原。世界地圖 15721 格，遙遙領先 —— 手冊：「最普遍的
	// 伊姆洛斯地形」。遭遇組含探險者、賊、郊狼、毒蛇、龍，也逐項對上。
	TerrainPlains Terrain = 1
	// TerrainSwamp 沼澤。**證據最硬的一個**：它是唯一掛到第 13 組的地形，
	// 而那一組就是鬼火（Will o wisp）與鬼墳族（Shambling mound）——
	// 手冊點名的正是這兩隻。另外「沒有類人族」「昆蟲很多」「不死很少」
	// 三條也都成立（沒有第 6 組、第 8 組出現兩次、第 0 組只出現一次）。
	TerrainSwamp Terrain = 2
	// TerrainHills 山丘。**唯一直接驗證的一個**：可通行性值 3 的兩個 tile
	// 就是移動要多算一步的那兩個，對上手冊「穿過本地形所花費的時間可能是
	// 一般地形的兩倍」。遭遇組是巨人與洞熊與龍，也對上手冊。
	TerrainHills Terrain = 3
	// TerrainDungeonA 地城地板。tile 0x00／0x13／0x53，MAP1 與 MAP5 的主要
	// 地板。遭遇組是不死／巨人／類人族／惡魔／蝙蝠／元素 —— 典型地城組合，
	// 手冊的七大地形裡沒有對應項。
	TerrainDungeonA Terrain = 4
	// TerrainTundra 凍土。單一 tile 0x5a（黑底白點＝雪），遭遇組第 15 組
	// 是雪人、雪巨人、冰龍、冰惡魔、極地熊、寒冬狼、冰元素。
	TerrainTundra Terrain = 5
	// TerrainDungeonB 地城地板（另一種）。單一 tile 0x56，MAP3 有 2598 格。
	// 遭遇組偏惡魔，同樣不在手冊的七大地形裡。
	TerrainDungeonB Terrain = 6
	// TerrainDesert 沙漠。遭遇組第 17 組是苦修僧（Dervish）與蜥蜴人
	// （Salamander）—— 手冊點名的正是這兩者。而且**八個槽位裡沒有探險者**，
	// 只有賊，對上手冊「除了些賊之外其他人類不願前來」。
	TerrainDesert Terrain = 7
)

// terrainNames 是給介面用的中文名。兩個地城類別不是手冊的地形。
var terrainNames = [NumTerrains]string{
	"森林", "平原", "沼澤", "山丘", "地城", "凍土", "地城", "沙漠",
}

// Name 回傳地形的中文名。
func (t Terrain) Name() string {
	if int(t) >= NumTerrains {
		return "未知"
	}
	return terrainNames[t]
}

// Outdoor 回報這是不是手冊講的野外地形（把兩個地城地板排除）。
func (t Terrain) Outdoor() bool {
	return t < NumTerrains && t != TerrainDungeonA && t != TerrainDungeonB
}

// EncounterEntry 是遭遇群組裡的一筆。
type EncounterEntry struct {
	// Monster 是 MONSTER.DAT 的索引。
	Monster int
	// Level 是這一筆的適用等級。**加了 100 的那種另有含意**，見 Gated。
	Level int
}

// Gated 回報這筆是不是「清一色群」。
//
// 等級值 >= 100 的那種（222f:2c9e）有兩個額外規則：只有前兩隻抽得到，
// 而且一旦抽中，這場遭遇剩下的每一隻都是同一種。沼澤那一組十筆全是這種，
// 所以出來的一定是清一色的鬼火群或鬼墳族群。
func (e EncounterEntry) Gated() bool { return e.Level >= encounterLevelGated }

// EffectiveLevel 回傳實際拿去比對的等級值（>= 100 的先減掉 100）。
func (e EncounterEntry) EffectiveLevel() int {
	if e.Gated() {
		return e.Level - encounterLevelGated
	}
	return e.Level
}

// Allowed 回報難度 level 能不能遇到這一筆。
//
// 原版（222f:2ca8–2cb8）：`level − 1 <= 這筆的等級 <= level + 1`。
// **這條對每一筆都成立**，不只是 Gated 的那種 —— 兩條路徑都落到同一段檢查。
// 效果是遭遇會跟著隊伍變強：難度 1 遇骷髏，難度 9 遇元素。
func (e EncounterEntry) Allowed(level int) bool {
	l := e.EffectiveLevel()
	return level >= l-1 && level <= l+1
}

// EncounterGroup 是一組會一起出現的怪物。
type EncounterGroup struct {
	// MinLevel／MaxLevel 是這一組適用的隊伍等級範圍（222f:2bde–2bef）。
	MinLevel, MaxLevel int
	// Entries 是十筆候選。同一隻重複出現代表權重較高。
	Entries [EncounterEntries]EncounterEntry
}

// Fits 回報隊伍等級落不落在這一組的範圍內。
func (g EncounterGroup) Fits(partyLevel int) bool {
	return partyLevel >= g.MinLevel && partyLevel <= g.MaxLevel
}

// TerrainGroups 回傳某個地形掛的 8 個遭遇群組編號。
//
// 原版擲 `rnd(8)` 從中挑一個；重複出現的群組機率較高
// （例如沙漠的 8 個槽位有 4 個是同一組沙漠怪）。
func (t *Tables) TerrainGroups(terrain Terrain) ([EncounterSlots]byte, error) {
	var out [EncounterSlots]byte
	if int(terrain) >= NumTerrains {
		return out, fmt.Errorf("gamedata: 地形 %d 超出範圍 0–%d",
			terrain, NumTerrains-1)
	}
	copy(out[:], t.terrainGroups[int(terrain)*EncounterSlots:])
	return out, nil
}

// EncounterGroup 取回一組遭遇。
func (t *Tables) EncounterGroup(i int) (EncounterGroup, error) {
	if i < 0 || i >= NumEncounterGroups {
		return EncounterGroup{}, fmt.Errorf("gamedata: 遭遇群組 %d 超出範圍 0–%d",
			i, NumEncounterGroups-1)
	}
	return t.encounters[i], nil
}

// NumEncounterGroupsLoaded 回傳實際解出幾組。
func (t *Tables) NumEncounterGroupsLoaded() int { return len(t.encounters) }

// parseEncounters 解出兩張遭遇表。
func parseEncounters(raw []byte) ([]byte, []EncounterGroup) {
	terrain := make([]byte, NumTerrains*EncounterSlots)
	copy(terrain, raw[offEncounterTerrain:offEncounterTerrain+len(terrain)])

	groups := make([]EncounterGroup, NumEncounterGroups)
	for i := range groups {
		b := raw[offEncounterGroups+i*encounterGroupSize:]
		g := EncounterGroup{MinLevel: int(b[0]), MaxLevel: int(b[1])}
		for j := 0; j < EncounterEntries; j++ {
			g.Entries[j] = EncounterEntry{
				Monster: int(b[2+2*j]),
				Level:   int(b[3+2*j]),
			}
		}
		groups[i] = g
	}
	return terrain, groups
}
