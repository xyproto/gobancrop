package gobancrop

import (
	"image"
	"image/color"
	"math"
	"sort"
)

type Point struct{ X, Y float64 }

type Quadrilateral [4]Point

// shrinkQuadAligned insets an axis-aligned quad by half a grid cell on all sides, trimming margins and labels.
func shrinkQuadAligned(q Quadrilateral) Quadrilateral {
	minX, minY := q[0].X, q[0].Y
	maxX, maxY := q[2].X, q[2].Y
	cell := (maxX - minX) / 18.0
	inset := cell * 0.5
	return Quadrilateral{
		{minX + inset, minY + inset},
		{maxX - inset, minY + inset},
		{maxX - inset, maxY - inset},
		{minX + inset, maxY - inset},
	}
}

func cropQuad(img *image.NRGBA, q Quadrilateral) *image.NRGBA {
	minX, minY := q[0].X, q[0].Y
	maxX, maxY := minX, minY
	for _, p := range q[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	r := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX)), int(math.Ceil(maxY))).Intersect(img.Bounds())
	return img.SubImage(r).(*image.NRGBA)
}

func interpQuadPoint(q Quadrilateral, u, v float64) Point {
	return Point{X: (1-v)*((1-u)*q[0].X+u*q[1].X) + v*((1-u)*q[3].X+u*q[2].X), Y: (1-v)*((1-u)*q[0].Y+u*q[1].Y) + v*((1-u)*q[3].Y+u*q[2].Y)}
}

func sampleBilinear(img *image.NRGBA, pt Point) color.Color {
	x, y := pt.X, pt.Y
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	x1, y1 := x0+1, y0+1
	fx, fy := x-float64(x0), y-float64(y0)
	c00, c10, c01, c11 := getSafe(img, x0, y0), getSafe(img, x1, y0), getSafe(img, x0, y1), getSafe(img, x1, y1)
	r00, g00, b00, a00 := c00.RGBA()
	r10, g10, b10, a10 := c10.RGBA()
	r01, g01, b01, a01 := c01.RGBA()
	r11, g11, b11, a11 := c11.RGBA()
	r0 := float64(r00)*(1-fx) + float64(r10)*fx
	r1 := float64(r01)*(1-fx) + float64(r11)*fx
	rf := r0*(1-fy) + r1*fy
	g0 := float64(g00)*(1-fx) + float64(g10)*fx
	g1 := float64(g01)*(1-fx) + float64(g11)*fx
	gf := g0*(1-fy) + g1*fy
	b0 := float64(b00)*(1-fx) + float64(b10)*fx
	b1 := float64(b01)*(1-fx) + float64(b11)*fx
	bf := b0*(1-fy) + b1*fy
	a0 := float64(a00)*(1-fx) + float64(a10)*fx
	a1 := float64(a01)*(1-fx) + float64(a11)*fx
	af := a0*(1-fy) + a1*fy
	return color.NRGBA{uint8(rf / 257), uint8(gf / 257), uint8(bf / 257), uint8(af / 257)}
}

func getSafe(img *image.NRGBA, x, y int) color.Color {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return color.NRGBA{0, 0, 0, 0}
	}
	return img.At(x, y)
}

func avgBrightness(c color.Color) uint32 {
	r, g, b, _ := c.RGBA()
	return (r + g + b) / 3
}

func rgbToHSV(r, g, b float64) (h, s, v float64) {
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	v, d := mx, mx-mn
	if mx > 0 {
		s = d / mx
	}
	switch {
	case d == 0:
		h = 0
	case mx == r:
		h = math.Mod((g-b)/d, 6) * 60
	case mx == g:
		h = ((b-r)/d + 2) * 60
	default:
		h = ((r-g)/d + 4) * 60
	}
	if h < 0 {
		h += 360
	}
	return
}

func brightnessHist(img *image.NRGBA, mask func(color.Color) bool) (hist [256]int, masked, total int) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 2 {
		for x := b.Min.X; x < b.Max.X; x += 2 {
			total++
			c := img.At(x, y)
			if !mask(c) {
				continue
			}
			masked++
			hist[avgBrightness(c)/257]++
		}
	}
	return
}

func otsu(hist [256]int, total int) int {
	var sumT, sumB, wB, maxV float64
	var thresh int
	for i, c := range hist {
		sumT += float64(i * c)
	}
	for t := 0; t < 256; t++ {
		wB += float64(hist[t])
		if wB == 0 {
			continue
		}
		wF := float64(total) - wB
		if wF == 0 {
			break
		}
		sumB += float64(t * hist[t])
		mB := sumB / wB
		mF := (sumT - sumB) / wF
		v := wB * wF * (mB - mF) * (mB - mF)
		if v > maxV {
			maxV = v
			thresh = t
		}
	}
	return thresh
}

func estimateDarkFrac(img *image.NRGBA, thr uint32) float64 {
	h := img.Bounds().Dy()
	col := img.Bounds().Dx() / 2
	var runs []int
	run := 0
	for y := 0; y < h; y++ {
		r, g, b, _ := img.At(col, y).RGBA()
		if (r+g+b)/3 < thr {
			run++
		} else if run > 0 {
			runs = append(runs, run)
			run = 0
		}
	}
	if run > 0 {
		runs = append(runs, run)
	}
	if len(runs) == 0 {
		return 0.02
	}
	sort.Ints(runs)
	return float64(runs[len(runs)/2]) / float64(h)
}

func autoSetup(img *image.NRGBA) (uint32, uint32, float64) {
	hist, m, _ := brightnessHist(img, func(color.Color) bool { return true })
	t := otsu(hist, m)
	f := estimateDarkFrac(img, uint32(t)*257)
	if f < 0.01 {
		f = 0.01
	}
	return uint32(t) * 257, uint32((t+255)/2) * 257, f
}

func scanSegments(limit, depth int, thr uint32, frac float64, maxW int,
	isDark func(int, int, uint32) bool, mask func(int, int) bool,
) [][2]int {
	minD := int(frac * float64(depth))
	var raw [][]int
	var curr []int
	for i := 0; i < limit; i++ {
		cnt := 0
		for j := 0; j < depth; j++ {
			if !mask(i, j) {
				continue
			}
			if isDark(i, j, thr) {
				cnt++
			}
		}
		if cnt >= minD {
			curr = append(curr, i)
		} else if len(curr) > 0 {
			raw = append(raw, curr)
			curr = nil
		}
	}
	if len(curr) > 0 {
		raw = append(raw, curr)
	}
	var segs [][2]int
	for _, s := range raw {
		if len(s) <= maxW {
			segs = append(segs, [2]int{s[0], s[len(s)-1]})
		}
	}
	return segs
}

func refineLines(segs [][2]int) []float64 {
	if len(segs) < 2 {
		return nil
	}
	mids := make([]float64, len(segs))
	for i, s := range segs {
		mids[i] = float64(s[0]+s[len(s)-1]) / 2
	}
	sort.Float64s(mids)
	start, end := mids[0], mids[len(mids)-1]
	step := (end - start) / 18
	lines := make([]float64, 19)
	for i := range lines {
		lines[i] = start + float64(i)*step
	}
	return lines
}

func isWood(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	rf, gf, bf := float64(r)/65535, float64(g)/65535, float64(b)/65535
	h, s, v := rgbToHSV(rf, gf, bf)
	return h >= 15 && h <= 50 && s >= 0.2 && v >= 0.2
}

func hueDelta(h1, h2 float64) float64 {
	d := math.Abs(h1 - h2)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func findLines(img *image.NRGBA, w, h int, thr uint32, _ float64) (ys, xs []float64) {
	woodHue := 35.0
	mask := func(_, _ int) bool { return true }

	isGridPixel := func(x, y int) bool {
		r, g, b, _ := img.At(x, y).RGBA()
		avg := (r + g + b) / 3
		hue, _, _ := rgbToHSV(float64(r)/65535, float64(g)/65535, float64(b)/65535)
		return hueDelta(hue, woodHue) > 25 || avg < 20000
	}

	fracs := []float64{0.03, 0.025, 0.02, 0.015, 0.01, 0.0075}
	widths := []int{8, 7, 6, 5, 4, 3}

	for _, frac := range fracs {
		for _, width := range widths {
			isDarkH := func(y, x int, _ uint32) bool { return isGridPixel(x, y) }
			isDarkV := func(x, y int, _ uint32) bool { return isGridPixel(x, y) }

			hs := scanSegments(h, w, thr, frac, width, isDarkH, mask)
			vs := scanSegments(w, h, thr, frac, width, isDarkV, mask)

			ysCand := refineLines(hs)
			xsCand := refineLines(vs)

			if len(ysCand) == 19 && len(xsCand) == 19 {
				return ysCand, xsCand
			}
		}
	}

	return nil, nil
}
