package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// debugScene 是具名的開發者書籤。它只決定開場地圖與座標；劇情旗標仍由
// -glyphs、-proving、-plot 等明確旗標控制，避免「跳到房間」偷偷替測試者
// 解掉謎題，讓正常／未解鎖分支無法驗證。
type debugScene struct {
	MapID int
	X     int
	Y     int
	// Facing 只在 SetFacing 為真時覆寫；其餘書籤保留存檔面向。
	Facing    game.Facing
	SetFacing bool
	Note      string
}

var debugScenes = map[string]debugScene{
	"new-game":       {MapID: 34, X: 28, Y: 50, Note: "新遊戲起點（只跳位置；完整建角請加 -newgame）"},
	"new-gleon":      {MapID: 34, X: 38, Y: 47, Note: "新格里昂附近／購船航路"},
	"enchanter":      {MapID: 2, X: 28, Y: 5, Note: "矮人大師附魔工坊"},
	"orb":            {MapID: 2, X: 42, Y: 28, Note: "恆世寶珠"},
	"demon-crystal":  {MapID: 4, X: 7, Y: 4, Note: "惡魔水晶"},
	"armory":         {MapID: 1, X: 24, Y: 27, Note: "兵器庫四座台座"},
	"cage-secret":    {MapID: 1, X: 13, Y: 26, Note: "鐵籠房間西牆密門"},
	"crushing-walls": {MapID: 1, X: 15, Y: 38, Note: "活動牆走廊入口"},
	"eregore":        {MapID: 1, X: 59, Y: 1, Note: "艾瑞戈爾觸發格前一步"},
	"cemetery":       {MapID: 3, X: 18, Y: 60, Note: "會重排的墓園入口"},
	"bell":           {MapID: 1, X: 26, Y: 8, Note: "夜鐘（可搭配 -hour=25）"},
	"travellers-bed": {MapID: 4, X: 21, Y: 57, Note: "旅人的床"},
	"void-riddle":    {MapID: 5, X: 11, Y: 19, Note: "幽靈司祭密語 VOID 前一步"},
	"jesric-riddle":  {MapID: 4, X: 59, Y: 38, Note: "馬利馮門房密語 JESRIC 前一步"},
	"circle-light":   {MapID: 5, X: 11, Y: 48, Note: "光之環入口（已解符印分支請加 -glyphs）"},
	// 站在出貨資料唯一一格「已注意」的水池陷阱東側。固定朝西讓 L／V
	// 可直接重播，不必先走一步而誤觸陷阱。
	"trap-pool": {
		MapID: 1, X: 12, Y: 17, Facing: game.West, SetFacing: true,
		Note: "水池陷阱東側；L 解除／V 觀室（技能請加 -give-skill）",
	},
}

func findDebugScene(name string) (debugScene, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	s, ok := debugScenes[key]
	if !ok {
		return debugScene{}, fmt.Errorf("未知場景 %q；可用 -list-scenes 查看名稱", name)
	}
	return s, nil
}

func debugSceneList() string {
	names := make([]string, 0, len(debugScenes))
	for name := range debugScenes {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		s := debugScenes[name]
		facing := ""
		if s.SetFacing {
			facing = fmt.Sprintf(" 面向%d", s.Facing)
		}
		fmt.Fprintf(&b, "%-16s 地圖%-2d (%2d,%2d)%s  %s\n",
			name, s.MapID, s.X, s.Y, facing, s.Note)
	}
	return b.String()
}
