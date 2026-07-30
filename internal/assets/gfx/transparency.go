package gfx

import (
	"image"
	"image/color"
)

// TransparentBackground 把與畫布邊界連通的 key 色背景改成透明，同時保留
// 緊貼非 key 像素的一圈 key 色輪廓。
//
// 老素材的步行人物是「彩色人物＋整格黑底」。直接把所有黑色清掉會挖空衣服、
// 盾牌與人物內部陰影；只清邊界連通背景又會把與背景相接的黑色外框一起吃掉。
// 因此先 flood-fill 找真正背景，再把貼著人物本體的一圈黑色留下。
func TransparentBackground(src image.Image, key color.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}

	w, h := b.Dx(), b.Dy()
	isKey := func(x, y int) bool {
		c := color.RGBAModel.Convert(dst.At(x, y)).(color.RGBA)
		return c.R == key.R && c.G == key.G && c.B == key.B && c.A == key.A
	}
	background := make([]bool, w*h)
	queue := make([]image.Point, 0, 2*(w+h))
	push := func(x, y int) {
		if x < 0 || x >= w || y < 0 || y >= h || background[y*w+x] || !isKey(x, y) {
			return
		}
		background[y*w+x] = true
		queue = append(queue, image.Pt(x, y))
	}
	for x := 0; x < w; x++ {
		push(x, 0)
		push(x, h-1)
	}
	for y := 0; y < h; y++ {
		push(0, y)
		push(w-1, y)
	}
	for head := 0; head < len(queue); head++ {
		p := queue[head]
		push(p.X-1, p.Y)
		push(p.X+1, p.Y)
		push(p.X, p.Y-1)
		push(p.X, p.Y+1)
	}

	nearBody := func(x, y int) bool {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx < 0 || nx >= w || ny < 0 || ny >= h || (dx == 0 && dy == 0) {
					continue
				}
				if !isKey(nx, ny) {
					return true
				}
			}
		}
		return false
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if background[y*w+x] && !nearBody(x, y) {
				c := dst.RGBAAt(x, y)
				c.A = 0
				dst.SetRGBA(x, y, c)
			}
		}
	}
	return dst
}
