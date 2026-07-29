package game

import "github.com/wicanr2/demon_winter_cht/internal/rng"

// 海戰使用 27×27 海域；原版把玩家船放在 (13,13)。
const (
	SeaSize   = 27
	SeaCentre = SeaSize / 2
	// SeaTurnPoints 是玩家船每回合的移動點。IDA `sub_1C427:1C472`
	// 直接把第一筆海戰單位的 +0x06 寫成 6。
	SeaTurnPoints = 6
)

type SeaKind byte

const (
	SeaPlayer SeaKind = iota
	SeaPirate
	SeaMonster
)

type SeaUnit struct {
	Name       string
	Kind       SeaKind
	X, Y       int
	Facing     Facing
	Hull       int
	MaxHull    int
	Experience int
}

func (u *SeaUnit) Alive() bool { return u != nil && u.Hull > 0 }

type SeaOutcome byte

const (
	SeaOngoing SeaOutcome = iota
	SeaVictory
	SeaSunk
	SeaEscaped
)

type SeaBattle struct {
	rng     *rng.RNG
	Units   []*SeaUnit
	Player  int
	Points  int
	Round   int
	Outcome SeaOutcome
}

func NewSeaBattle(r *rng.RNG, hull int, playerName string, enemies []*SeaUnit) *SeaBattle {
	if hull < 1 {
		hull = 1
	}
	player := &SeaUnit{Name: playerName, Kind: SeaPlayer, X: SeaCentre, Y: SeaCentre,
		Facing: North, Hull: hull, MaxHull: ShipMaxHull}
	units := []*SeaUnit{player}
	units = append(units, enemies...)
	return &SeaBattle{rng: r, Units: units, Points: SeaTurnPoints, Round: 1}
}

func (b *SeaBattle) PlayerShip() *SeaUnit { return b.Units[b.Player] }

func (b *SeaBattle) Spend(cost int) bool {
	if b.Outcome != SeaOngoing || cost < 0 || b.Points < cost {
		return false
	}
	b.Points -= cost
	return true
}

func (b *SeaBattle) Turn(delta int) bool {
	if !b.Spend(2) {
		return false
	}
	u := b.PlayerShip()
	u.Facing = Facing((int(u.Facing) + delta + 4) % 4)
	return true
}

func seaDelta(f Facing) (int, int) {
	switch f {
	case North:
		return 0, -1
	case East:
		return 1, 0
	case South:
		return 0, 1
	default:
		return -1, 0
	}
}

func (b *SeaBattle) occupied(x, y int, except *SeaUnit) bool {
	for _, u := range b.Units {
		if u != except && u.Alive() && u.X == x && u.Y == y {
			return true
		}
	}
	return false
}

func (b *SeaBattle) Move(reverse bool) bool {
	cost := 1
	sign := 1
	if reverse {
		cost, sign = 3, -1
	}
	if !b.Spend(cost) {
		return false
	}
	u := b.PlayerShip()
	dx, dy := seaDelta(u.Facing)
	nx, ny := u.X+dx*sign, u.Y+dy*sign
	if b.occupied(nx, ny, u) {
		return true // 原版仍花掉本次移動點。
	}
	u.X, u.Y = nx, ny
	if nx < 0 || ny < 0 || nx >= SeaSize || ny >= SeaSize {
		b.Outcome = SeaEscaped
	}
	return true
}

type CannonResult struct {
	Fired, Hit, Sunk bool
	Target           *SeaUnit
	Damage, Distance int
}

// Fire 朝指定絕對方向開砲。命中率隨距離遞減；未命中時砲彈會偏到
// 相鄰平行線，仍可能誤擊別艘船，對應手冊明載的 friendly/other hit。
func (b *SeaBattle) Fire(dir Facing) CannonResult {
	if !b.Spend(3) {
		return CannonResult{}
	}
	p := b.PlayerShip()
	dx, dy := seaDelta(dir)
	target, dist := b.firstOnRay(p.X, p.Y, dx, dy, 0)
	if target == nil {
		return CannonResult{Fired: true}
	}
	chance := 100 - dist*8
	if chance < 20 {
		chance = 20
	}
	if b.rng.Roll(100) > chance {
		offset := -1
		if b.rng.Roll(2) == 2 {
			offset = 1
		}
		target, dist = b.firstOnRay(p.X, p.Y, dx, dy, offset)
		if target == nil {
			return CannonResult{Fired: true}
		}
	}
	damage := b.rng.Roll(10)
	target.Hull -= damage
	if target.Hull < 0 {
		target.Hull = 0
	}
	res := CannonResult{Fired: true, Hit: true, Sunk: !target.Alive(),
		Target: target, Damage: damage, Distance: dist}
	b.refreshOutcome()
	return res
}

func (b *SeaBattle) firstOnRay(x, y, dx, dy, offset int) (*SeaUnit, int) {
	ox, oy := -dy*offset, dx*offset
	for d := 1; d < SeaSize; d++ {
		tx, ty := x+dx*d+ox, y+dy*d+oy
		for _, u := range b.Units {
			if u.Alive() && u.X == tx && u.Y == ty {
				return u, d
			}
		}
	}
	return nil, 0
}

// EnemyTurn 依序驅動敵人。海盜在同列/同行時開砲，否則逼近；
// 海怪只會逼近並在相鄰時撞擊。兩者都由手冊直接規定。
func (b *SeaBattle) EnemyTurn() []CannonResult {
	var out []CannonResult
	p := b.PlayerShip()
	for _, u := range b.Units {
		if u.Kind == SeaPlayer || !u.Alive() || b.Outcome != SeaOngoing {
			continue
		}
		dx, dy := p.X-u.X, p.Y-u.Y
		if seaAbs(dx)+seaAbs(dy) == 1 {
			damage := b.rng.Roll(10)
			p.Hull -= damage
			if p.Hull < 0 {
				p.Hull = 0
			}
			out = append(out, CannonResult{Fired: u.Kind == SeaPirate, Hit: true,
				Target: p, Damage: damage, Distance: 1, Sunk: !p.Alive()})
			b.refreshOutcome()
			continue
		}
		if u.Kind == SeaPirate && (dx == 0 || dy == 0) {
			dist := seaAbs(dx) + seaAbs(dy)
			chance := 100 - dist*8
			if chance < 20 {
				chance = 20
			}
			res := CannonResult{Fired: true, Distance: dist}
			if b.rng.Roll(100) <= chance {
				res.Hit, res.Target, res.Damage = true, p, b.rng.Roll(10)
				p.Hull -= res.Damage
				if p.Hull < 0 {
					p.Hull = 0
				}
				res.Sunk = !p.Alive()
			}
			out = append(out, res)
			b.refreshOutcome()
			continue
		}
		stepX, stepY := 0, 0
		if seaAbs(dx) >= seaAbs(dy) {
			stepX = seaSign(dx)
		} else {
			stepY = seaSign(dy)
		}
		if !b.occupied(u.X+stepX, u.Y+stepY, u) {
			u.X += stepX
			u.Y += stepY
		}
	}
	if b.Outcome == SeaOngoing {
		b.Round++
		b.Points = SeaTurnPoints
	}
	return out
}

func (b *SeaBattle) refreshOutcome() {
	if !b.PlayerShip().Alive() {
		b.Outcome = SeaSunk
		return
	}
	for _, u := range b.Units {
		if u.Kind != SeaPlayer && u.Alive() {
			return
		}
	}
	b.Outcome = SeaVictory
}

func (b *SeaBattle) Experience() int {
	total := 0
	for _, u := range b.Units {
		if u.Kind != SeaPlayer && !u.Alive() {
			total += u.Experience
		}
	}
	return total
}

func seaAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func seaSign(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}
