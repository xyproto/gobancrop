package gobancrop

import (
	"image"
	"image/color"
	"math"
)

type Point struct{ X, Y float64 }
type Quadrilateral [4]Point

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
	return Point{
		X: (1-v)*((1-u)*q[0].X+u*q[1].X) + v*((1-u)*q[3].X+u*q[2].X),
		Y: (1-v)*((1-u)*q[0].Y+u*q[1].Y) + v*((1-u)*q[3].Y+u*q[2].Y),
	}
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
