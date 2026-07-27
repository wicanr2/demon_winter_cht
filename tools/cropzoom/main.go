// 從截圖切一塊放大，並列出出現的顏色與佔比 —— 拿實機畫面當 oracle 時用。
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"sort"
)

func main() {
	in := flag.String("in", "", "")
	out := flag.String("out", "", "")
	x0 := flag.Int("x", 0, "")
	y0 := flag.Int("y", 0, "")
	w := flag.Int("w", 100, "")
	h := flag.Int("h", 100, "")
	zoom := flag.Int("zoom", 6, "")
	flag.Parse()

	f, err := os.Open(*in)
	if err != nil { panic(err) }
	src, err := png.Decode(f)
	if err != nil { panic(err) }
	f.Close()

	dst := image.NewRGBA(image.Rect(0, 0, *w**zoom, *h**zoom))
	count := map[[3]uint32]int{}
	for y := 0; y < *h; y++ {
		for x := 0; x < *w; x++ {
			c := src.At(*x0+x, *y0+y)
			r, g, b, _ := c.RGBA()
			count[[3]uint32{r >> 8, g >> 8, b >> 8}]++
			for zy := 0; zy < *zoom; zy++ {
				for zx := 0; zx < *zoom; zx++ {
					dst.Set(x**zoom+zx, y**zoom+zy, c)
				}
			}
		}
	}
	o, err := os.Create(*out)
	if err != nil { panic(err) }
	defer o.Close()
	png.Encode(o, dst)

	type kv struct{ c [3]uint32; n int }
	var list []kv
	for c, n := range count { list = append(list, kv{c, n}) }
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	fmt.Printf("切出 %dx%d，放大 %dx → %s\n顏色分佈（前 12）：\n", *w, *h, *zoom, *out)
	for i, e := range list {
		if i >= 12 { break }
		fmt.Printf("  #%02X%02X%02X  %5d px (%4.1f%%)\n", e.c[0], e.c[1], e.c[2], e.n,
			100*float64(e.n)/float64(*w**h))
	}
}
