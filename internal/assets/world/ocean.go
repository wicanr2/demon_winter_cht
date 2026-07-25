package world

// 海面的兩種浪花圖塊。
//
// 原版載入地圖時會把一部分 OceanTile 隨機換成 OceanTileAlt，讓大片海洋
// 看起來有變化。兩者畫出來都是橫向浪紋，差別只在 OceanTile 摻了白色浪沫。
//
// **這是純外觀的替換**：兩個 tile 在 FILES.DAT 可通行性表裡的值都是 0xff
// （不可步行的海面），所以換不換不影響任何遊戲規則。這一點是本套件把它
// 放在算繪端、而不是寫進 Map 資料裡的依據 —— 原版是在載入時就地改寫地圖
// 緩衝區，但既然改的結果不參與規則判定，保持 Map 是「檔案解出來的原樣」
// 比較好驗證。`TestOceanDither_PurelyCosmetic` 守著這個前提。
const (
	OceanTile    = 0x14
	OceanTileAlt = 0x62
)

// 原版的擲點器：一個乘法同餘產生器，反組譯自 15be:179f
// （DEMON.INT 檔位移 0x1af7f）。
//
//	1af85  mov ax,0x7d          ; 125
//	1af88  mul word [ds:0x2f4c] ; 種子 *= 125（取低 16 位）
//	1af8c  mov [ds:0x2f4c],ax
//	1af8f  cmp ax,0x7fff
//	1af92  jbe → 回 0
//	1af94       回 1            ; 最高位是 1 就回 1
//
// 也就是「乘 125、看最高位」的公平硬幣。種子在資料段 DS:0x2f4c
// （檔位移 0x28a4c）的初值是 0x2711 = 10001。
//
// 這個產生器與 internal/rng 那個（浮點 LCG，管遊戲規則的擲點）**不是同一個**。
// 這裡刻意不共用：這條只決定海面畫哪一格，混用會讓規則面的亂數序列被畫面
// 影響，兩邊都變得不可重現。
const (
	oceanLCGMultiplier = 125
	oceanLCGSeed       = 10001
)

// OceanDither 是海面浪花的擲點器。零值不可用，用 NewOceanDither 建立。
type OceanDither struct {
	state uint16
}

// NewOceanDither 以原版的初始種子建立擲點器。
func NewOceanDither() *OceanDither {
	return &OceanDither{state: oceanLCGSeed}
}

// NewOceanDitherSeed 以指定種子建立擲點器。
//
// 原版的種子是全域的，跨地圖累積推進 —— 同一張地圖在不同時機載入，浪花
// 圖案就不一樣。所以「與原版逐格相同」本來就不是原版自己做得到的事，
// 這裡開放種子只是為了讓測試與截圖可重現。
func NewOceanDitherSeed(seed uint16) *OceanDither {
	return &OceanDither{state: seed}
}

// Next 推進一次並回報這一格要不要換成 OceanTileAlt。
func (d *OceanDither) Next() bool {
	d.state *= oceanLCGMultiplier
	return d.state >= 0x8000
}

// Apply 就地把 tiles 裡的海面格隨機換成另一種浪花。
//
// 只碰 OceanTile，其餘一律不動；每碰到一格就推進一次擲點器，
// 與原版的走訪順序（線性掃過整張地圖，見 15be:17c0 = 檔位移 0x1afa0）一致。
func (d *OceanDither) Apply(tiles []byte) {
	for i, v := range tiles {
		if v != OceanTile {
			continue
		}
		if d.Next() {
			tiles[i] = OceanTileAlt
		}
	}
}
