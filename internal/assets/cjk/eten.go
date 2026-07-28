// Package cjk 讀取倚天中文系統（ETEN）的原生點陣字，供中文化渲染使用。
//
// 為什麼用倚天而不是把 TTF rasterize：1990s DOS 中文遊戲的中文長什麼樣，
// 倚天就長什麼樣。TTF 縮到 16px 會糊、筆劃比例也不對；倚天是為這個尺寸
// 手工調的點陣。
//
// **字型檔不隨本專案散布**（倚天是有版權的商業軟體），
// 由使用者自備並以路徑指定，與原版遊戲資料同一個處理方式。
package cjk

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"golang.org/x/text/encoding/traditionalchinese"
)

// 16 點字型的點陣規格。每列 (W+7)/8 bytes、MSB-first、由上而下。
const (
	GlyphWidth  = 16
	GlyphHeight = 15
	rowBytes    = (GlyphWidth + 7) / 8
	glyphStride = rowBytes * GlyphHeight // 30
)

// Big5 分區索引的邊界。**不是線性索引** —— 符號區與漢字區分屬兩個檔案，
// 常用字與次常用字之間還有一段空隙要跳過。
//
// 驗證 oracle：STDFONT 的 idx 0 必須是「一」（一條橫線）。
// 先過這關再往下做，否則整批字會整體偏移，看起來像「有字但都不對」。
var (
	lastSpc    = big5Raw(0xA3, 0xBF) // 符號區尾 = 407
	baseA440   = big5Raw(0xA4, 0x40) // 漢字區起點
	lastCommon = big5Raw(0xC6, 0x7E) // 常用字尾
	baseC940   = big5Raw(0xC9, 0x40) // 次常用起點
)

// nCommon 是常用字的數量。
const nCommon = 5401

// big5Raw 把 Big5 雙位元組換成線性序號。
func big5Raw(hi, lo byte) int {
	off := int(lo) - 0x40
	if lo >= 0x7F {
		off = int(lo) - 0x62
	}
	return (int(hi)-0xA1)*157 + off
}

// manualBig5 補上 Go 的 Big5 編碼器對不上的字。
//
// 少數全形符號的 Unicode 對應在 codec 與 Big5 表之間有歧義
// （例如 ～ 是 U+FF5E 還是 U+301C），編碼會失敗，用手動表補。
var manualBig5 = map[rune][2]byte{
	'～': {0xA1, 0xE3},
}

// Font 是載入好的倚天點陣字。
type Font struct {
	std []byte // STDFONT.15：漢字區
	spc []byte // SPCFONT.15：全形符號／標點
	bold bool   // 每列向右膨脹 1px，保留原生 16×15 的手工字形
}

// Options 控制倚天字模的顯示方式。
type Options struct {
	// Bold 以 MI2 中文化相同的方式橫向加粗一像素。
	Bold bool
}

// Load 讀取倚天 16 點字型。
//
// **一定要同時帶 SPCFONT.15。** STDFONT 從 A440（「一」）起，
// 不含 A140–A3BF 的全形標點；只帶 STDFONT 會讓
// 「，。！？「」『』（）《》」全部落到 fallback，
// 畫面上「字是倚天、標點是另一種字」很突兀。
func Load(stdPath, spcPath string) (*Font, error) {
	return LoadWithOptions(stdPath, spcPath, Options{})
}

// LoadWithOptions 讀取倚天 16×15 點陣字型並套用顯示選項。
func LoadWithOptions(stdPath, spcPath string, opts Options) (*Font, error) {
	std, err := readFontFile(stdPath)
	if err != nil {
		return nil, err
	}
	spc, err := readFontFile(spcPath)
	if err != nil {
		return nil, err
	}
	f := &Font{std: std, spc: spc, bold: opts.Bold}

	// 索引公式的自我檢查：idx 0 必須是「一」—— 只有第 7 列有連續橫線。
	if !f.looksLikeYi() {
		return nil, fmt.Errorf("cjk: %s 的第 0 個字不像「一」，索引公式或檔案版本不符", stdPath)
	}
	return f, nil
}

// LoadDir 從目錄載入字型。倚天光碟使用大寫檔名，但 Linux 上常見的
// 解壓工具會轉成小寫，因此兩種拼法都接受。
func LoadDir(dir string, opts Options) (*Font, error) {
	std, err := fontPath(dir, "STDFONT.15", "stdfont.15")
	if err != nil {
		return nil, err
	}
	spc, err := fontPath(dir, "SPCFONT.15", "spcfont.15")
	if err != nil {
		return nil, err
	}
	return LoadWithOptions(std, spc, opts)
}

func fontPath(dir string, names ...string) (string, error) {
	for _, name := range names {
		path := filepath.Join(dir, name)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("cjk: %s 缺少 %s", dir, names[0])
}

func readFontFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cjk: 讀取字型 %s 失敗: %w", path, err)
	}
	if len(data) == 0 || len(data)%glyphStride != 0 {
		return nil, fmt.Errorf("cjk: %s 長度 %d 不是 %d 的整數倍，格式不符",
			path, len(data), glyphStride)
	}
	return data, nil
}

// looksLikeYi 檢查 STDFONT 的第 0 個字是不是「一」。
//
// 判準刻意寬鬆：只要求「恰有一列的前 15 個像素幾乎全亮，其餘列幾乎全暗」。
// 這足以擋掉整體偏移，又不會因字型版本的細微差異而誤判。
func (f *Font) looksLikeYi() bool {
	if len(f.std) < glyphStride {
		return false
	}
	dense := 0
	for y := 0; y < GlyphHeight; y++ {
		w := uint16(f.std[y*2])<<8 | uint16(f.std[y*2+1])
		if bitsSet(w) >= 12 {
			dense++
		}
	}
	return dense == 1
}

func bitsSet(w uint16) int {
	n := 0
	for ; w != 0; w &= w - 1 {
		n++
	}
	return n
}

// standardBig5 是「Unicode → 標準 Big5 碼位」的對照表，由解碼方向反建。
//
// **不能只用 Go 的 Big5 編碼器。** 它對一批字會挑到標準區以外的碼位：
// 簡體字被映進造字區（`头`→`8960`），而 `撐` 這種標準繁體字會被編成
// 倚天延伸區的 `FCB9`，儘管它在標準區 `BCB9` 就有字模。
// 兩種情況的結果一樣 —— 字型查不到，畫面上一片空白。
//
// 反建對照表能一次解決整類問題：把標準區每一個碼位解碼成 Unicode，
// 反過來當成查表的唯一依據。碼位由小到大掃，重複的字取較小的碼位
// （Big5 有少數字重複收錄，較小的那個才是常用碼位）。
var standardBig5 = buildStandardBig5()

func buildStandardBig5() map[rune][2]byte {
	dec := traditionalchinese.Big5.NewDecoder()
	m := make(map[rune][2]byte, 14000)

	for hi := 0xA1; hi <= 0xF9; hi++ {
		for lo := 0x40; lo <= 0xFE; lo++ {
			if lo > 0x7E && lo < 0xA1 {
				continue
			}
			out, err := dec.Bytes([]byte{byte(hi), byte(lo)})
			if err != nil {
				continue
			}
			rs := []rune(string(out))
			// 未定義的碼位會解成替換字元，不是真的有字。
			if len(rs) != 1 || rs[0] == '�' {
				continue
			}
			if _, dup := m[rs[0]]; dup {
				continue
			}
			m[rs[0]] = [2]byte{byte(hi), byte(lo)}
		}
	}
	return m
}

// Big5 把一個字元編成標準 Big5 雙位元組。
// 回傳 ok=false 表示這個字不在標準 Big5 裡，沒有對應字模。
//
// **一定要驗編出來的碼落在標準區間。** Go 的 Big5 編碼器會把一批簡體字
// 映進造字區（`头`→`8960`、`马`→`89C6`、`着`→`FED3`），編碼「成功」
// 但倚天字型完全不涵蓋那些碼位 —— 只看 err 會讓簡體字一路過關，
// 到畫面上才發現是一片空白。
func Big5(ch rune) (hi, lo byte, ok bool) {
	if b, hit := manualBig5[ch]; hit {
		return b[0], b[1], true
	}
	b, hit := standardBig5[ch]
	if !hit {
		return 0, 0, false
	}
	return b[0], b[1], true
}

// Glyph 取出一個字的點陣。回傳 ok=false 表示這個字不在 Big5 或字型範圍內，
// 呼叫端應改用 fallback 字型。
//
// **fallback 數量是品質指標**：若一大批字掉進 fallback，
// 先懷疑索引公式或漏帶 SPCFONT，不要無腦補字型。
func (f *Font) Glyph(ch rune) (*image.Alpha, bool) {
	hi, lo, ok := Big5(ch)
	if !ok {
		return nil, false
	}

	src, idx := f.locate(hi, lo)
	if src == nil {
		return nil, false
	}
	off := idx * glyphStride
	if off < 0 || off+glyphStride > len(src) {
		return nil, false
	}

	img := image.NewAlpha(image.Rect(0, 0, GlyphWidth, GlyphHeight))
	for y := 0; y < GlyphHeight; y++ {
		w := uint16(src[off+y*2])<<8 | uint16(src[off+y*2+1])
		if f.bold {
			w = emboldenRow(w)
		}
		for x := 0; x < GlyphWidth; x++ {
			if w>>(15-x)&1 == 1 {
				img.SetAlpha(x, y, color.Alpha{A: 0xff})
			}
		}
	}
	return img, true
}

// emboldenRow 與 MI2 的 16×15 粗體實作一致：原列與向右移一格的副本 OR。
// 以數值表示時，MSB 是畫面最左像素，所以向右移是 >> 1。
func emboldenRow(row uint16) uint16 {
	return row | row>>1
}

// locate 依 Big5 分區決定這個字在哪個檔案的第幾個位置。
func (f *Font) locate(hi, lo byte) ([]byte, int) {
	r := big5Raw(hi, lo)
	switch {
	case r < 0:
		return nil, 0
	case r <= lastSpc:
		return f.spc, r
	case r <= lastCommon:
		return f.std, r - baseA440
	default:
		return f.std, nCommon + (r - baseC940)
	}
}

// GlyphCount 回傳兩個字型檔各自的字數，供載入後檢查。
func (f *Font) GlyphCount() (std, spc int) {
	return len(f.std) / glyphStride, len(f.spc) / glyphStride
}
