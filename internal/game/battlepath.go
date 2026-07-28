package game

// FirstStepToward 在目前戰場地形與佔位下，找一條走到 target 相鄰格的
// 最短路徑，回傳第一步。它只規劃，不移動單位也不花行動點。
//
// preferred 只決定同長路徑的搜尋順序；直走不通時依順時針、逆時針、
// 反向嘗試。這讓呼叫端的動作穩定可重播，同時允許先繞開隊友或凹形地形。
func (b *Battle) FirstStepToward(u, target *Unit, preferred Facing) (Facing, bool) {
	if b == nil || u == nil || target == nil || !u.Alive() || !target.Alive() {
		return preferred, false
	}
	if manhattan(u.X, u.Y, target.X, target.Y) == 1 {
		return preferred, false
	}

	type node struct {
		x, y  int
		first Facing
	}
	directions := []Facing{preferred, preferred.CW(), preferred.CCW(), preferred.Reverse()}
	seen := map[[2]int]bool{{u.X, u.Y}: true}
	queue := make([]node, 0, BattleFieldSize*BattleFieldSize)

	enqueue := func(x, y int, first Facing) {
		p := [2]int{x, y}
		if seen[p] {
			return
		}
		seen[p] = true
		queue = append(queue, node{x: x, y: y, first: first})
	}

	for _, dir := range directions {
		probe := *u
		probe.Facing = int(dir)
		if !b.CanStep(&probe) {
			continue
		}
		dx, dy := dir.Delta()
		enqueue(u.X+dx, u.Y+dy, dir)
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if manhattan(cur.x, cur.y, target.X, target.Y) == 1 {
			return cur.first, true
		}
		for _, dir := range directions {
			probe := *u
			probe.X, probe.Y, probe.Facing = cur.x, cur.y, int(dir)
			if !b.CanStep(&probe) {
				continue
			}
			dx, dy := dir.Delta()
			enqueue(cur.x+dx, cur.y+dy, cur.first)
		}
	}
	return preferred, false
}

func manhattan(ax, ay, bx, by int) int {
	dx, dy := ax-bx, ay-by
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}
