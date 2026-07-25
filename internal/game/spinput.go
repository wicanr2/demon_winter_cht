package game

// SPInput 是「投入多少法力」的輸入模型。
//
// 原版施法時會問「How many S.P.」—— 投得多效果強，但法力用完就沒了，
// 所以這是個真的要玩家決定的數字，不是確認框。
//
// 這裡刻意只放純邏輯、不碰按鍵也不碰畫面：呈現層在 cmd/demonwinter，
// 而那個套件匯入 Ebiten，Ebiten 在 init 期就要求顯示器，寫在那邊的東西
// 在無頭環境下連測都測不了（同 internal/ui/layout 的理由）。
type SPInput struct {
	minSP, maxSP int
	amount       int

	// typing 為真代表玩家已經開始打數字。第一次按數字要清掉預設值重打，
	// 不然「預設 19、想打 5」會接成 195。
	typing bool
}

// NewSPInput 建立輸入模型，預設填滿目前法力 —— 多數情況玩家就是要全投。
//
// maxSP 小於 minSP（法力不足）時兩者都取 minSP；呼叫端本來就該先擋掉
// 法力不足的情況，這裡只保證不會生出上界小於下界的區間。
func NewSPInput(minSP, maxSP int) *SPInput {
	if maxSP < minSP {
		maxSP = minSP
	}
	return &SPInput{minSP: minSP, maxSP: maxSP, amount: maxSP}
}

// Min／Max 是可投入的範圍（含端點）。
func (s *SPInput) Min() int { return s.minSP }
func (s *SPInput) Max() int { return s.maxSP }

// Amount 是目前輸入的值。它永遠落在 [Min, Max] 之內。
func (s *SPInput) Amount() int {
	s.clamp()
	return s.amount
}

func (s *SPInput) clamp() {
	if s.amount < s.minSP {
		s.amount = s.minSP
	}
	if s.amount > s.maxSP {
		s.amount = s.maxSP
	}
}

// Adjust 把值加減 delta（方向鍵用）。
func (s *SPInput) Adjust(delta int) {
	s.amount += delta
	s.typing = false
	s.clamp()
}

// AppendDigit 把一個 0–9 的數字接到目前輸入的後面。
//
// 接出來超過上限就忽略那一鍵 —— 無聲截斷成別的數字比沒反應更難懂。
func (s *SPInput) AppendDigit(digit int) {
	if digit < 0 || digit > 9 {
		return
	}
	if !s.typing {
		s.amount = 0
		s.typing = true
	}
	if next := s.amount*10 + digit; next <= s.maxSP {
		s.amount = next
	}
}

// Backspace 刪掉最後一位。
func (s *SPInput) Backspace() {
	s.amount /= 10
	s.typing = true
}
