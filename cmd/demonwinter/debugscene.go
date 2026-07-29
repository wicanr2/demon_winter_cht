package main

import (
	"fmt"
	"sort"
	"strings"
)

// debugScene 是具名的開發者書籤。它只決定開場地圖與座標；劇情旗標仍由
// -glyphs、-proving、-plot 等明確旗標控制，避免「跳到房間」偷偷替測試者
// 解掉謎題，讓正常／未解鎖分支無法驗證。
type debugScene struct {
	MapID int
	X     int
	Y     int
	Note  string
}

var debugScenes = map[string]debugScene{
	"new-game":       {34, 28, 50, "新遊戲起點（只跳位置；完整建角請加 -newgame）"},
	"new-gleon":      {34, 38, 47, "新格里昂附近／購船航路"},
	"enchanter":      {2, 28, 5, "矮人大師附魔工坊"},
	"orb":            {2, 42, 28, "恆世寶珠"},
	"demon-crystal":  {4, 7, 4, "惡魔水晶"},
	"armory":         {1, 24, 27, "兵器庫四座台座"},
	"cage-secret":    {1, 13, 26, "鐵籠房間西牆密門"},
	"crushing-walls": {1, 15, 38, "活動牆走廊入口"},
	"eregore":        {1, 59, 1, "艾瑞戈爾觸發格前一步"},
	"cemetery":       {3, 18, 60, "會重排的墓園入口"},
	"bell":           {1, 26, 8, "夜鐘（可搭配 -hour=25）"},
	"travellers-bed": {4, 21, 57, "旅人的床"},
	"void-riddle":    {5, 11, 19, "幽靈司祭密語 VOID 前一步"},
	"jesric-riddle":  {4, 59, 38, "馬利馮門房密語 JESRIC 前一步"},
	"circle-light":   {5, 11, 48, "光之環入口（已解符印分支請加 -glyphs）"},
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
		fmt.Fprintf(&b, "%-16s 地圖%-2d (%2d,%2d)  %s\n",
			name, s.MapID, s.X, s.Y, s.Note)
	}
	return b.String()
}
