package gobancrop

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
)

const maxLineWidth = 5

// FindGoban finds the coarse, axis-aligned board bounding box by wood-color.
func FindGoban(img *image.NRGBA) (Quadrilateral, error) {
	log.Printf("FindGoban: scan bounds %v", img.Bounds())
	b := img.Bounds()
	minX, minY := float64(b.Max.X), float64(b.Max.Y)
	maxX, maxY := 0.0, 0.0
	found := false
	isWood := func(c color.Color) bool {
		r, g, b, _ := c.RGBA()
		rf, gf, bf := float64(r)/65535, float64(g)/65535, float64(b)/65535
		h, s, _ := rgbToHSV(rf, gf, bf)
		return h >= 20 && h <= 45 && s >= 0.3
	}
	for y := b.Min.Y; y < b.Max.Y; y += 2 {
		for x := b.Min.X; x < b.Max.X; x += 2 {
			if isWood(img.At(x, y)) {
				found = true
				xF, yF := float64(x), float64(y)
				if xF < minX {
					minX = xF
				}
				if xF > maxX {
					maxX = xF
				}
				if yF < minY {
					minY = yF
				}
				if yF > maxY {
					maxY = yF
				}
			}
		}
	}
	if !found {
		return Quadrilateral{}, errors.New("no wood region found")
	}
	q := Quadrilateral{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
	log.Printf("FindGoban: bounds %v", q)
	return q, nil
}

// FindActualBoard fits the 19×19 grid within a coarse quad, returning a refined quad.
func FindActualBoard(img *image.NRGBA, quad Quadrilateral) (Quadrilateral, error) {
	log.Printf("FindActualBoard: input %v", quad)
	sub := cropQuad(img, quad)
	w, h := sub.Bounds().Dx(), sub.Bounds().Dy()
	log.Printf("subimage %dx%d", w, h)
	thr, _, darkFrac := autoSetup(sub)
	log.Printf("thr=%d darkFrac=%.3f", thr, darkFrac)
	ys, xs := findLines(sub, w, h, thr, darkFrac)
	log.Printf("lines h=%d v=%d", len(ys), len(xs))
	if len(ys) != 19 || len(xs) != 19 {
		return Quadrilateral{}, fmt.Errorf("grid not found: h=%d v=%d", len(ys), len(xs))
	}
	tl := interpQuadPoint(quad, xs[0]/float64(w-1), ys[0]/float64(h-1))
	tr := interpQuadPoint(quad, xs[18]/float64(w-1), ys[0]/float64(h-1))
	br := interpQuadPoint(quad, xs[18]/float64(w-1), ys[18]/float64(h-1))
	bl := interpQuadPoint(quad, xs[0]/float64(w-1), ys[18]/float64(h-1))
	r := Quadrilateral{tl, tr, br, bl}
	log.Printf("FindActualBoard: refined %v", r)
	return r, nil
}

// CropAndCorrect warps the quad to a size×size square, perspective-corrected.
func CropAndCorrect(img *image.NRGBA, quad Quadrilateral, size int) (*image.NRGBA, error) {
	log.Printf("CropAndCorrect: size=%d quad=%v", size, quad)
	if size <= 0 {
		return nil, errors.New("invalid size")
	}
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		v := float64(y) / float64(size-1)
		for x := 0; x < size; x++ {
			u := float64(x) / float64(size-1)
			src := Point{
				X: (1-v)*((1-u)*quad[0].X+u*quad[1].X) + v*((1-u)*quad[3].X+u*quad[2].X),
				Y: (1-v)*((1-u)*quad[0].Y+u*quad[1].Y) + v*((1-u)*quad[3].Y+u*quad[2].Y),
			}
			out.Set(x, y, sampleBilinear(img, src))
		}
	}
	log.Print("CropAndCorrect: done")
	return out, nil
}
