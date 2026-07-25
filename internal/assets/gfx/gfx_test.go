package gfx

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// 測試資料根目錄：workplace/orig/demwin/DEM_DATA（唯讀，不可修改）。
// 找不到就整批 skip——這是刻意設計，讓沒有原版素材的環境（如 CI）也能跑
// go test 不報錯，但也就驗不到「真的解對了沒有」，本地一定要跑一次。
func origDataDir(t *testing.T) string {
	t.Helper()
	// gfx_test.go 位在 internal/assets/gfx/，往上 3 層到 repo root。
	dir := filepath.Join("..", "..", "..", "workplace", "orig", "demwin", "DEM_DATA")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("找不到原版素材目錄 %s，skip(唯讀資料不入版控，需先解壓 Demons Winter (1988).zip 到 workplace/orig/)", dir)
	}
	return dir
}

func dumpDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "workplace", "dump", "gfx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func readAsset(t *testing.T, dir, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Skipf("讀不到 %s: %v", name, err)
	}
	return data
}

// --- CGA 全螢幕圖：OPEN.PIC ---
// 已驗證(肉眼比對 workplace/dosbox/shots/05-cga-hang-open-pic.png 完全吻合):
// 320x200、線性排列(非硬體偶奇交錯)、2bpp、palette1 高亮度。
func TestDecodeOpenPIC(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "OPEN.PIC")
	img, err := DecodeCGALinear320(data, 320, 200)
	if err != nil {
		t.Fatalf("解碼失敗: %v", err)
	}
	out := filepath.Join(dumpDir(t), "open-pic-cga-linear.png")
	if err := SavePNG(out, img); err != nil {
		t.Fatalf("存 PNG 失敗: %v", err)
	}
	t.Logf("輸出 %s，已與 workplace/dosbox/shots/05-cga-hang-open-pic.png 肉眼比對吻合", out)
}

// 對照組:硬體偶奇交錯版本,驗證「交錯」是錯的假設(應該是花的/位置錯亂)。
func TestDecodeOpenPICInterleavedControl(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "OPEN.PIC")
	img, err := DecodeCGAPlanar320(data, 320, 200)
	if err != nil {
		t.Fatalf("解碼失敗: %v", err)
	}
	out := filepath.Join(dumpDir(t), "open-pic-cga-interleaved-control.png")
	if err := SavePNG(out, img); err != nil {
		t.Fatalf("存 PNG 失敗: %v", err)
	}
	t.Logf("輸出 %s(對照組,已驗證是錯的假設/花的)", out)
}

// --- CGA 人像/場景框：PIC1.PIC 等 5,184 B 檔 ---
// 假設:144x144,線性排列(比照 OPEN.PIC 已驗證的佈局)。
func TestDecodeCGAPortraits(t *testing.T) {
	dir := origDataDir(t)
	names := []string{"PIC1.PIC", "PIC2.PIC", "PIC3.PIC", "PIC4.PIC", "PIC5.PIC", "PIC6.PIC", "PRIEST.PIC", "SHAMEN.PIC", "THANATOS.PIC"}
	for _, name := range names {
		data := readAsset(t, dir, name)
		img, err := DecodeCGALinear320(data, 144, 144)
		if err != nil {
			t.Fatalf("%s 解碼失敗: %v", name, err)
		}
		out := filepath.Join(dumpDir(t), "cga-portrait-"+name+".png")
		if err := SavePNG(out, img); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
	}
	t.Logf("輸出 %d 張人像/場景框 PNG 到 %s", len(names), dumpDir(t))
}

// --- EGA 人像/場景框：PIC1.PIE 等 18,160 B 檔 ---
// 假設:開頭 16 bytes 調色盤索引 + 144x252(144*1.75) 4-plane sequential。
func TestDecodeEGAPortraits(t *testing.T) {
	dir := origDataDir(t)
	names := []string{"PIC1.PIE", "PIC2.PIE", "PIC3.PIE", "PIC4.PIE", "PIC5.PIE", "PIC6.PIE", "PRIEST.PIE", "SHAMEN.PIE", "THANATOS.PIE"}
	for _, name := range names {
		data := readAsset(t, dir, name)
		pal, rest, err := ParsePIEPalette(data)
		if err != nil {
			t.Fatalf("%s 調色盤解析失敗: %v", name, err)
		}
		img, err := DecodeEGAPlanar(rest, 144, 252, EGAPlanesSequential, pal)
		if err != nil {
			t.Fatalf("%s 解碼失敗(用內嵌調色盤): %v", name, err)
		}
		out := filepath.Join(dumpDir(t), "ega-portrait-"+name+"-withpal.png")
		if err := SavePNG(out, img); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
		// 對照組:不用內嵌調色盤索引,直接用標準 16 色盤(驗證內嵌 16 bytes
		// 到底是不是調色盤——如果兩張圖幾乎一樣,代表這 16 bytes 其實是
		// identity mapping 或根本不是調色盤)。
		imgStd, err := DecodeEGAPlanar(rest, 144, 252, EGAPlanesSequential, nil)
		if err != nil {
			t.Fatalf("%s 解碼失敗(標準調色盤): %v", name, err)
		}
		outStd := filepath.Join(dumpDir(t), "ega-portrait-"+name+"-stdpal.png")
		if err := SavePNG(outStd, imgStd); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
	}
	t.Logf("輸出 %d 組人像/場景框 PNG(內嵌調色盤版+標準調色盤對照版)到 %s", len(names), dumpDir(t))
}

// --- CGA 精靈圖 sprite sheet ---
// 已驗證：frame 一律 16x16、64 bytes，**含 CYPHER.SHP**。
//
// 這不是靠位元組數推的 —— 遊戲自己在視訊模式初始化時宣告了 frame 大小：
// CGA 分支 `1d9f:018d MOV word ptr [0x5226],0x40`（64 bytes/frame，
// 2bpp 下即 16x16），並直接用它乘出各素材檔大小
// （`x0x66` = 64*102 = 6528 = DEMON.SHP、`x0x1b` = 64*27 = 1728 = CYPHER.SHP）。
//
// 先前記載的 16x32 / CYPHER 8x32 是錯的：位元組數在兩種讀法下都整除，
// 算術分不出來，而 16x32 解出來的每個 frame 其實是「兩個完整圖形上下疊著」。
func TestDecodeCGASprites(t *testing.T) {
	dir := origDataDir(t)
	cases := []struct {
		name           string
		frameW, frameH int
	}{
		{"COMBAT.SHP", 16, 16},
		{"SHIP.SHP", 16, 16},
		{"DEMON.SHP", 16, 16},
		{"WINTER.SHP", 16, 16},
		{"MONSTER.SHP", 16, 16},
		{"CYPHER.SHP", 16, 16},
	}
	for _, c := range cases {
		data := readAsset(t, dir, c.name)
		frames, err := DecodeCGASpriteSheet(data, c.frameW, c.frameH)
		if err != nil {
			t.Fatalf("%s 解碼失敗(frame %dx%d): %v", c.name, c.frameW, c.frameH, err)
		}
		cols := 10
		sheet := TileSpriteSheet(frames, cols)
		out := filepath.Join(dumpDir(t), "cga-sprites-"+c.name+".png")
		if err := SavePNG(out, sheet); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
		t.Logf("%s: %d frames, %dx%d each -> %s", c.name, len(frames), c.frameW, c.frameH, out)
	}
}

// --- EGA 精靈圖 sprite sheet ---
// 已驗證：**檔案內** frame 是 16x28、224 bytes，全部六個 .SHE 都一樣
// （CYPHER 不是特例）。plane 佈局 EGAPlanesRowBlocks 維持不變。
//
// FUN_217b_07cf 硬編碼的 frame stride 0x1c0=448 沒有錯，但它描述的是
// **載入時做過水平 pixel doubling 之後、記憶體緩衝區裡**的 frame，
// 不是磁碟格式：
//
//	1d9f:00bf  MOV word ptr [0x5226],0xe0   ; 224 = 檔案內 frame 大小
//	1d9f:0101  SHL AX,0x1
//	1d9f:0109  MOV [0x521a],AX              ; 448 = 記憶體內 frame 大小
//
// 載入器 FUN_1d9f_0a8b 只對副檔名 "shE"/"SHE" 呼叫 FUN_217b_0adf 就地加倍，
// 每 byte 逐 bit 複製兩次。加倍後結構同構（每列 4 個 plane 各 2->4 bytes），
// 所以「檔案當 16x28 解」與「記憶體當 32x28 解」是同一張圖。
//
// 用 32x28 解檔案會看到「一格裡 2x2 四個小圖」—— 那是兩個 frame 左右各半
// 錯位疊起來的假象，不是美術打包。
func TestDecodeEGASprites(t *testing.T) {
	dir := origDataDir(t)
	cases := []struct {
		name           string
		frameW, frameH int
	}{
		{"COMBAT.SHE", 16, 28},
		{"SHIP.SHE", 16, 28},
		{"DEMON.SHE", 16, 28},
		{"WINTER.SHE", 16, 28},
		{"MONSTER.SHE", 16, 28},
	}
	for _, c := range cases {
		data := readAsset(t, dir, c.name)
		frames, err := DecodeEGASpriteSheet(data, c.frameW, c.frameH, EGAPlanesRowBlocks)
		if err != nil {
			t.Fatalf("%s 解碼失敗(frame %dx%d): %v", c.name, c.frameW, c.frameH, err)
		}
		cols := 10
		sheet := TileSpriteSheet(frames, cols)
		out := filepath.Join(dumpDir(t), "ega-sprites-"+c.name+".png")
		if err := SavePNG(out, sheet); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
		t.Logf("%s: %d frames, %dx%d each -> %s", c.name, len(frames), c.frameW, c.frameH, out)

		// 放大單一 frame(第 0 張)方便肉眼細看、跟 CGA 版對照。
		if len(frames) > 0 {
			zoomed := zoomImage(frames[0], 8)
			zoomOut := filepath.Join(dumpDir(t), "zoom-ega-"+c.name+"-frame0.png")
			_ = SavePNG(zoomOut, zoomed)
			t.Logf("%s frame0 放大 -> %s", c.name, zoomOut)
		}
	}
}

// CYPHER.SHE 與其他五個 .SHE 完全同規則：16x28、224 bytes/frame、27 個 frame。
// 它曾被當成「唯一 224 bytes 的特例」，其實反了 —— 224 才是常態，
// 它只是因為 frame 數 27 是奇數，才是唯一無法被（記憶體側的）448 整除的檔。
// 這支測試的參數一直都是對的，是六個檔裡唯一沒解錯的。
func TestDecodeEGASpriteCypher(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "CYPHER.SHE")
	frames, err := DecodeEGASpriteSheet(data, 16, 28, EGAPlanesRowBlocks)
	if err != nil {
		t.Fatalf("解碼失敗: %v", err)
	}
	out := filepath.Join(dumpDir(t), "ega-sprites-CYPHER.SHE.png")
	if err := SavePNG(out, TileSpriteSheet(frames, 10)); err != nil {
		t.Fatalf("存 PNG 失敗: %v", err)
	}
	t.Logf("CYPHER.SHE(16x28): %d frames -> %s", len(frames), out)
}

// 交叉比對用：CGA MONSTER.SHP frame0(16x16)放大 8 倍，
// 拿來跟 EGA MONSTER.SHE frame0(16x28)並排肉眼核對是不是同一隻怪物。
func TestDecodeCGAMonsterFrame0Zoom(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "MONSTER.SHP")
	frames, err := DecodeCGASpriteSheet(data, 16, 16)
	if err != nil {
		t.Fatalf("解碼失敗: %v", err)
	}
	zoomed := zoomImage(frames[0], 8)
	out := filepath.Join(dumpDir(t), "zoom-cga-MONSTER.SHP-frame0-correct.png")
	if err := SavePNG(out, zoomed); err != nil {
		t.Fatalf("存 PNG 失敗: %v", err)
	}
	t.Logf("CGA frame0(16x16,已驗證) -> %s", out)
}

// 對照組：「逐-frame 四平面各自整塊 sequential」佈局在正確的 16x28 尺寸下
// 依然是雜訊 —— 留著證明 EGAPlanesRowBlocks 不是碰巧猜中的。
func TestDecodeEGASpritesPerFrameControl(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "MONSTER.SHE")
	frames, err := DecodeEGASpriteSheet(data, 16, 28, EGAPlanesSequential)
	if err != nil {
		t.Fatalf("解碼失敗: %v", err)
	}
	out := filepath.Join(dumpDir(t), "ega-sprites-MONSTER.SHE-perframe-control.png")
	if err := SavePNG(out, TileSpriteSheet(frames, 10)); err != nil {
		t.Fatalf("存 PNG 失敗: %v", err)
	}
	t.Logf("對照組(已驗證是錯的假設/雜訊) -> %s", out)
}

// --- OPEN.PIE：不符 3.5x 規律的異常檔案，多組候選假設探索 ---
func TestDecodeOpenPIEExploration(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "OPEN.PIE")
	t.Logf("OPEN.PIE 大小 = %d bytes", len(data))

	// 候選 1: 去掉 16 byte 調色盤 header 後，當單張 640x319-ish 4-plane 圖
	// (102160-16)/4 = 25536 = width/8 * height，25536 沒有 5 的因數，
	// 320/640 寬(rowBytes 含 5 的因數 40/80)都除不盡，此候選預期失敗，
	// 只是留下紀錄用來排除。
	pal, rest, err := ParsePIEPalette(data)
	if err == nil {
		attempts := []struct{ w, h int }{
			{320, 200}, {320, 350}, {640, 350}, {320, 399}, {608, 336}, {456, 448},
		}
		for _, a := range attempts {
			img, derr := DecodeEGAPlanar(rest, a.w, a.h, EGAPlanesSequential, pal)
			if derr != nil {
				t.Logf("候選 %dx%d 失敗(尺寸不合): %v", a.w, a.h, derr)
				continue
			}
			out := filepath.Join(dumpDir(t), "open-pie-try-"+itoa(a.w)+"x"+itoa(a.h)+".png")
			if serr := SavePNG(out, img); serr == nil {
				t.Logf("候選 %dx%d 輸出 %s", a.w, a.h, out)
			}
		}
	}

	// 候選 2: 當成 5~6 幀動畫，每幀套用「人像框」的 144x252 4-plane 公式
	// (18144 bytes/frame,不含 palette header),看是否有任何 offset 對得上。
	frameBytes := 18144
	maxFrames := len(data) / frameBytes
	for i := 0; i < maxFrames; i++ {
		chunk := data[i*frameBytes : (i+1)*frameBytes]
		img, derr := DecodeEGAPlanar(chunk, 144, 252, EGAPlanesSequential, nil)
		if derr != nil {
			continue
		}
		out := filepath.Join(dumpDir(t), "open-pie-frame144x252-"+itoa(i)+".png")
		_ = SavePNG(out, img)
	}
	t.Logf("OPEN.PIE 是探索性測試,不斷言解碼正確,結論見 docs/formats/graphics.md")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// assertCGAFontLayout 用「字形本身」而不是「解碼器沒報錯」來驗 CGA 佈局。
//
// 三條互相獨立的斷言：
//  1. `A`(0x41) 的 8x8 圖樣要真的長得像 A（原始 16 bytes:
//     03c0 0ff0 3c3c 3c3c 3ffc 3c3c 3c3c 0000）。
//     這條同時卡死「每字 16 bytes」「每列 2 bytes」「packed 2bpp、
//     byte0=左 4px / byte1=右 4px、MSB-first」「字表起於 0x20」四件事——
//     只要其中一項判錯，圖樣就對不上。
//  2. 空白(0x20)整格是背景。
//  3. bank0 只用色號 0/3（白字黑底）、bank1 只用色號 0/2（黑字亮洋紅底），
//     且兩者形狀互補——對應 1d9f:12ce 用 0xAA（四個像素全是色號 2）
//     填反白底色。
func assertCGAFontLayout(t *testing.T, glyphs []*image.RGBA) {
	t.Helper()
	// '#' = 前景（bank0 的色號 3），'.' = 背景（色號 0）
	wantA := []string{
		"...##...",
		"..####..",
		".##..##.",
		".##..##.",
		".######.",
		".##..##.",
		".##..##.",
		"........",
	}
	gA, ok := GlyphForChar(glyphs, 'A')
	if !ok {
		t.Fatalf("CGA 字型取不到 'A'")
	}
	for y, row := range wantA {
		for x := 0; x < cgaGlyphWidth; x++ {
			fg := gA.RGBAAt(x, y) == CGAPalette1High[3]
			if fg != (row[x] == '#') {
				t.Fatalf("CGA 'A' 圖樣不符（第 %d 列第 %d 欄）：解出來的字形不是 A，佈局判錯了", y, x)
			}
		}
	}
	gSpace, ok := GlyphForChar(glyphs, ' ')
	if !ok {
		t.Fatalf("CGA 字型取不到空白")
	}
	for y := 0; y < cgaGlyphHeight; y++ {
		for x := 0; x < cgaGlyphWidth; x++ {
			if gSpace.RGBAAt(x, y) != CGAPalette1High[0] {
				t.Fatalf("CGA 空白字元(0x20)在 (%d,%d) 不是背景色", x, y)
			}
		}
	}
	for i := 0; i < CGAFontBankGlyphs; i++ {
		norm, inv := glyphs[i], glyphs[CGAFontBankGlyphs+i]
		for y := 0; y < cgaGlyphHeight; y++ {
			for x := 0; x < cgaGlyphWidth; x++ {
				n, v := norm.RGBAAt(x, y), inv.RGBAAt(x, y)
				if n != CGAPalette1High[0] && n != CGAPalette1High[3] {
					t.Fatalf("bank0 glyph %d 在 (%d,%d) 用到色號 0/3 以外的顏色", i, x, y)
				}
				if v != CGAPalette1High[0] && v != CGAPalette1High[2] {
					t.Fatalf("bank1 glyph %d 在 (%d,%d) 用到色號 0/2 以外的顏色", i, x, y)
				}
				// 形狀互補：bank0 的前景(3) 對應 bank1 的前景(0)。
				if (n == CGAPalette1High[3]) != (v == CGAPalette1High[0]) {
					t.Fatalf("bank1 glyph %d 在 (%d,%d) 不是 bank0 的反白版", i, x, y)
				}
			}
		}
	}
}

// --- 字型 ---
// 已驗證（見 docs/re/17-font-format.md）：反組譯 FUN_217b_025a（CGA 路徑）
// 與 FUN_217b_097c（EGA 路徑）逐指令核對出真實佈局——
// CGA 8x8 packed 2bpp（每 byte 4 個像素、每列 2 bytes、來源線性、無 header、
// 兩個 96 字 bank：一般 + 反白）、EGA 16x14 1bpp。肉眼比對：整個 ASCII
// 字母表清楚可讀，GOT.FNE 呈現花體(blackletter)風格，跟
// workplace/dosbox/shots/smoke-01.png 主選單、03-ega-ingame.png
// 遊戲內選單的花體字一致。
//
// 2026-07-25 修正：CGA 一度被判成「1-byte header + 同列 2 byte 是 bit0/bit1
// 雙平面」，輸出是雜訊；重讀 217b:025a 與 1d9f:0f1e 的原始指令後改為上述
// packed 2bpp 才解出可讀字形（舊斷言已作廢，不要再套用）。
func TestDecodeFonts(t *testing.T) {
	dir := origDataDir(t)
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	black := color.RGBA{0x00, 0x00, 0x00, 0xff}

	// --- CGA ASC.FNT：8x8、packed 2bpp、兩個 bank ---
	ascFnt := readAsset(t, dir, "ASC.FNT")
	cgaGlyphs, err := DecodeCGAFont(ascFnt)
	if err != nil {
		t.Fatalf("ASC.FNT 解碼失敗: %v", err)
	}
	t.Logf("ASC.FNT: %d 個 glyph（2 個 bank × %d 字，從 0x20 起）", len(cgaGlyphs), CGAFontBankGlyphs)
	if want := 2 * CGAFontBankGlyphs; len(cgaGlyphs) != want {
		t.Fatalf("ASC.FNT glyph 數 = %d，預期 %d（3072 bytes / 16 = 192，尾端 0x1A 是 DOS EOF 不算資料）", len(cgaGlyphs), want)
	}
	assertCGAFontLayout(t, cgaGlyphs)
	atlasCGA := TileSpriteSheet(cgaGlyphs, 16)
	outAtlas := filepath.Join(dumpDir(t), "font-asc-fnt-atlas.png")
	if err := SavePNG(outAtlas, atlasCGA); err != nil {
		t.Fatalf("存 PNG 失敗: %v", err)
	}
	t.Logf("ASC.FNT atlas -> %s", outAtlas)
	zoomCGA := zoomImage(atlasCGA, 4)
	outZoom := filepath.Join(dumpDir(t), "font-asc-fnt-atlas-zoom4x.png")
	if err := SavePNG(outZoom, zoomCGA); err != nil {
		t.Fatalf("存 PNG 失敗: %v", err)
	}
	t.Logf("ASC.FNT atlas 放大4倍 -> %s", outZoom)

	// --- EGA ASC.FNE / GOT.FNE：16x14、1bpp ---
	for _, name := range []string{"ASC.FNE", "GOT.FNE"} {
		data := readAsset(t, dir, name)
		glyphs, err := DecodeEGAFont(data, white, black)
		if err != nil {
			t.Fatalf("%s 解碼失敗: %v", name, err)
		}
		t.Logf("%s: %d 個字元(從 0x20 起)", name, len(glyphs))
		atlas := TileSpriteSheet(glyphs, 16)
		out := filepath.Join(dumpDir(t), "font-"+name+"-atlas.png")
		if err := SavePNG(out, atlas); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
		t.Logf("%s atlas -> %s", name, out)
		zoom := zoomImage(atlas, 3)
		outZ := filepath.Join(dumpDir(t), "font-"+name+"-atlas-zoom3x.png")
		if err := SavePNG(outZ, zoom); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
		t.Logf("%s atlas 放大3倍 -> %s", name, outZ)
	}

	// --- 直接渲染實際 UI 字串，跟 DOSBox 截圖並排比對 ---
	// smoke-01.png 主選單、03-ega-ingame.png 遊戲內選單都是花體(GOT.FNE)。
	gotData := readAsset(t, dir, "GOT.FNE")
	gotGlyphs, err := DecodeEGAFont(gotData, white, black)
	if err != nil {
		t.Fatalf("GOT.FNE 解碼失敗(渲染字串用): %v", err)
	}
	strings := []string{
		"DEMON'S WINTER",
		"Go adventuring",
		"Character Utilities",
		"Alternate Character Set",
		"Walk",
		"Party info",
		"Save Game",
		"Camp",
	}
	for i, s := range strings {
		img := RenderText(gotGlyphs, s, egaGlyphWidth, egaGlyphHeight, black)
		out := filepath.Join(dumpDir(t), "font-render-got-"+itoa(i)+".png")
		if err := SavePNG(out, img); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
		t.Logf("GOT.FNE 渲染 %q -> %s", s, out)
	}

	// 同一批字串也用 ASC.FNE(平頭字)渲染一份對照，供肉眼確認兩種字型
	// 的差異(GOT=花體、ASC=平頭)不是解碼錯誤造成的假差異。
	ascData := readAsset(t, dir, "ASC.FNE")
	ascGlyphs, err := DecodeEGAFont(ascData, white, black)
	if err != nil {
		t.Fatalf("ASC.FNE 解碼失敗(渲染字串用): %v", err)
	}
	for i, s := range strings {
		img := RenderText(ascGlyphs, s, egaGlyphWidth, egaGlyphHeight, black)
		out := filepath.Join(dumpDir(t), "font-render-asc-"+itoa(i)+".png")
		if err := SavePNG(out, img); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
		t.Logf("ASC.FNE 渲染 %q -> %s", s, out)
	}
}

// 放大單一 frame 方便肉眼細看(縮圖版排列在一起容易誤判成「像形狀」，
// 這支測試把 COMBAT.SHP/MONSTER.SHP 各挑幾個 frame 放大 8 倍單獨輸出，
// 誠實檢查到底是不是真的雜訊)。
func TestDecodeCGASpritesZoom(t *testing.T) {
	dir := origDataDir(t)
	for _, name := range []string{"COMBAT.SHP", "MONSTER.SHP", "CYPHER.SHP"} {
		data := readAsset(t, dir, name)
		frames, err := DecodeCGASpriteSheet(data, 16, 16)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, idx := range []int{0, 1, 2, 3} {
			if idx >= len(frames) {
				continue
			}
			zoomed := zoomImage(frames[idx], 8)
			out := filepath.Join(dumpDir(t), "zoom-"+name+"-frame"+itoa(idx)+".png")
			_ = SavePNG(out, zoomed)
			t.Logf("%s frame %d -> %s", name, idx, out)
		}
	}
}
