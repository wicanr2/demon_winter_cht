package gfx

import (
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
// 已驗證(肉眼看 MONSTER.SHP frame0 是清楚的人形武士剪影，見
// docs/formats/graphics.md):frame 是「窄高」16x32(不是原先猜的 32x16 —
// 寬高猜反了，byte 數量剛好一樣所以除法對得上但畫面全花，肉眼比對才抓到)，
// CYPHER.SHP 更窄是 8x32。
func TestDecodeCGASprites(t *testing.T) {
	dir := origDataDir(t)
	cases := []struct {
		name           string
		frameW, frameH int
	}{
		{"COMBAT.SHP", 16, 32},
		{"SHIP.SHP", 16, 32},
		{"DEMON.SHP", 16, 32},
		{"WINTER.SHP", 16, 32},
		{"MONSTER.SHP", 16, 32},
		{"CYPHER.SHP", 8, 32},
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
// 已驗證（2026-07-25，見 docs/re/07-sprite-blit.md）：直接反組譯
// FUN_217b_07cf（217b 段 EGA 單一 sprite blit 常式）讀出真實佈局是
// 32x28、EGAPlanesRowBlocks（frame 內逐列排列、每列 4 個 plane 各自連續
// 一塊 rowBytes)，不是先前用位元組數整除猜測的 16x56 或整檔四平面分塊。
// COMBAT/SHIP/DEMON/WINTER/MONSTER 都是 448 bytes/frame，跟 07cf/06f9
// 硬編碼的 frame stride 0x1c0=448 完全對上。CYPHER.SHE 是 224 bytes/frame
// （非 448），不受 07cf 直接驗證，用同一種佈局公式外推 16x28（見下方
// TestDecodeEGASpriteCypher 與 docs/re/07-sprite-blit.md 的「假設」標記）。
func TestDecodeEGASprites(t *testing.T) {
	dir := origDataDir(t)
	cases := []struct {
		name           string
		frameW, frameH int
	}{
		{"COMBAT.SHE", 32, 28},
		{"SHIP.SHE", 32, 28},
		{"DEMON.SHE", 32, 28},
		{"WINTER.SHE", 32, 28},
		{"MONSTER.SHE", 32, 28},
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

// CYPHER.SHE frame 是 224 bytes(不是 448)，FUN_217b_07cf 的硬編碼 stride
// 0x1c0=448 不適用，這裡只是用同一個 EGAPlanesRowBlocks 公式外推
// 16x28(16/8=2 bytes/row * 28 rows * 4 planes = 224)，**假設**，未經
// 反組譯直接證實，見 docs/re/07-sprite-blit.md。
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
	t.Logf("CYPHER.SHE(假設 16x28): %d frames -> %s", len(frames), out)
}

// 交叉比對用：CGA MONSTER.SHP frame0(已驗證正確的 16x32)放大 8 倍，
// 拿來跟 EGA MONSTER.SHE frame0(32x28)並排肉眼核對是不是同一隻怪物。
func TestDecodeCGAMonsterFrame0Zoom(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "MONSTER.SHP")
	frames, err := DecodeCGASpriteSheet(data, 16, 32)
	if err != nil {
		t.Fatalf("解碼失敗: %v", err)
	}
	zoomed := zoomImage(frames[0], 8)
	out := filepath.Join(dumpDir(t), "zoom-cga-MONSTER.SHP-frame0-correct.png")
	if err := SavePNG(out, zoomed); err != nil {
		t.Fatalf("存 PNG 失敗: %v", err)
	}
	t.Logf("CGA frame0(16x32,已驗證) -> %s", out)
}

// 對照組：先前試過的「逐-frame 四平面各自整塊 sequential」在 32x28 尺寸下
// 一樣是雜訊(已排除的假設，尺寸猜對佈局仍猜錯的案例，留著讓人對比)。
func TestDecodeEGASpritesPerFrameControl(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "MONSTER.SHE")
	frames, err := DecodeEGASpriteSheet(data, 32, 28, EGAPlanesSequential)
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

// --- 字型 ---
// 已驗證（2026-07-25，見 docs/re/17-font-format.md）：反組譯 FUN_217b_025a
// （CGA 路徑）與 FUN_217b_097c（EGA 路徑）逐指令核對出真實佈局——
// CGA 8x8 2bpp「同列 2 個 byte 是 bit0/bit1 兩個平面」（不是先前假設的
// 8x12 1bpp VGA BIOS 格式）、EGA 16x14 1bpp。肉眼比對：整個 ASCII
// 字母表清楚可讀，GOT.FNE 呈現花體(blackletter)風格，跟
// workplace/dosbox/shots/smoke-01.png 主選單、03-ega-ingame.png
// 遊戲內選單的花體字一致。
func TestDecodeFonts(t *testing.T) {
	dir := origDataDir(t)
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	black := color.RGBA{0x00, 0x00, 0x00, 0xff}

	// --- CGA ASC.FNT：8x8、2bpp 逐列雙平面 ---
	ascFnt := readAsset(t, dir, "ASC.FNT")
	cgaGlyphs, err := DecodeCGAFont(ascFnt)
	if err != nil {
		t.Fatalf("ASC.FNT 解碼失敗: %v", err)
	}
	t.Logf("ASC.FNT: %d 個字元(從 0x20 起)", len(cgaGlyphs))
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

// 探索性:MONSTER.SHE 用不同佈局假設(row-interleaved planes、寬高对調)
// 快速掃描,找哪個(如果有)才是清楚的怪物剪影。
func TestDecodeEGASpritesLayoutSweep(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "MONSTER.SHE")

	// 1) 逐-frame row-interleaved planes, 32x28
	if frames, err := DecodeEGASpriteSheet(data, 32, 28, EGAPlanesRowInterleaved); err == nil {
		out := filepath.Join(dumpDir(t), "sweep-monster-rowinterleaved-32x28.png")
		_ = SavePNG(out, TileSpriteSheet(frames, 10))
		t.Logf("sweep 1 -> %s", out)
	} else {
		t.Logf("sweep 1 失敗: %v", err)
	}

	// 2) 寬高對調:56x16 (寬度套 1.75x, 高度維持 CGA 的 16)
	if frames, err := DecodeEGASpriteSheetGlobalPlanes(data, 56, 16); err == nil {
		out := filepath.Join(dumpDir(t), "sweep-monster-global-56x16.png")
		_ = SavePNG(out, TileSpriteSheet(frames, 10))
		t.Logf("sweep 2 -> %s", out)
	} else {
		t.Logf("sweep 2 失敗: %v", err)
	}

	// 3) 高度不變:32x16(2bpp->4bpp僅色深加倍,不做 1.75x 縮放),全域四平面
	if frames, err := DecodeEGASpriteSheetGlobalPlanes(data, 32, 16); err == nil {
		out := filepath.Join(dumpDir(t), "sweep-monster-global-32x16.png")
		_ = SavePNG(out, TileSpriteSheet(frames, 10))
		t.Logf("sweep 3 -> %s", out)
	} else {
		t.Logf("sweep 3 失敗: %v", err)
	}
}

// 放大單一 frame 方便肉眼細看(縮圖版排列在一起容易誤判成「像形狀」，
// 這支測試把 COMBAT.SHP/MONSTER.SHP 各挑幾個 frame 放大 8 倍單獨輸出，
// 誠實檢查到底是不是真的雜訊)。
func TestDecodeCGASpritesZoom(t *testing.T) {
	dir := origDataDir(t)
	for _, name := range []string{"COMBAT.SHP", "MONSTER.SHP", "CYPHER.SHP"} {
		w, h := 32, 16
		if name == "CYPHER.SHP" {
			w = 16
		}
		data := readAsset(t, dir, name)
		frames, err := DecodeCGASpriteSheet(data, w, h)
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

// 探索性:CGA sprite frame 尺寸猜測掃描(byte 數固定 128,寬高比未定)。
func TestDecodeCGASpritesDimensionSweep(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "MONSTER.SHP")
	for _, d := range []struct{ w, h int }{{32, 16}, {16, 32}, {8, 64}, {64, 8}} {
		frames, err := DecodeCGASpriteSheet(data, d.w, d.h)
		if err != nil {
			t.Logf("%dx%d 失敗: %v", d.w, d.h, err)
			continue
		}
		z := zoomImage(frames[0], 8)
		out := filepath.Join(dumpDir(t), "sweep-cga-monster-"+itoa(d.w)+"x"+itoa(d.h)+"-frame0.png")
		_ = SavePNG(out, z)
		t.Logf("%dx%d frame0 -> %s", d.w, d.h, out)
	}
}

// 探索性:MONSTER.SHE 用「已驗證的正確方向」16x56,搭配不同 plane 佈局假設。
func TestDecodeEGASpritesLayoutSweep2(t *testing.T) {
	dir := origDataDir(t)
	data := readAsset(t, dir, "MONSTER.SHE")

	if frames, err := DecodeEGASpriteSheet(data, 16, 56, EGAPlanesSequential); err == nil {
		out := filepath.Join(dumpDir(t), "sweep2-monster-perframe-seq-16x56.png")
		_ = SavePNG(out, TileSpriteSheet(frames, 10))
		t.Logf("perframe-seq -> %s", out)
	} else {
		t.Logf("perframe-seq 失敗: %v", err)
	}

	if frames, err := DecodeEGASpriteSheet(data, 16, 56, EGAPlanesRowInterleaved); err == nil {
		out := filepath.Join(dumpDir(t), "sweep2-monster-perframe-rowint-16x56.png")
		_ = SavePNG(out, TileSpriteSheet(frames, 10))
		t.Logf("perframe-rowint -> %s", out)
	} else {
		t.Logf("perframe-rowint 失敗: %v", err)
	}

	// global sequential 已經在 TestDecodeEGASprites 跑過(16x56),這裡補
	// global row-interleaved 對照:整檔視為一張大圖,但每列 4 個 plane byte 相鄰。
	rowBytes := 16 / 8
	totalHeight := len(data) / 4 / rowBytes
	if img, err := DecodeEGAPlanar(data, 16, totalHeight, EGAPlanesRowInterleaved, nil); err == nil {
		out := filepath.Join(dumpDir(t), "sweep2-monster-global-rowint-16xtall.png")
		_ = SavePNG(out, img)
		t.Logf("global-rowint -> %s (%dx%d)", out, 16, totalHeight)
	} else {
		t.Logf("global-rowint 失敗: %v", err)
	}
}
