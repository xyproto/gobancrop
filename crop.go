package gobancrop

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/xyproto/palgen"
)

func FindGoban(img *image.NRGBA) (Quadrilateral, error) {
	log.Printf("FindGoban: scan bounds %v", img.Bounds())
	img = reducePalette(img, 6)
	b := img.Bounds()
	minX, minY := float64(b.Max.X), float64(b.Max.Y)
	maxX, maxY := 0.0, 0.0
	found := false
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
		return Quadrilateral{}, fmt.Errorf("FindGoban: no wood region found")
	}
	q := Quadrilateral{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
	log.Printf("FindGoban: bounds %v", q)
	return q, nil
}

func reducePalette(img image.Image, n int) *image.NRGBA {
	reduced, err := palgen.Reduce(img, n)
	if err != nil {
		log.Printf("reducePalette: error reducing palette: %v", err)
		return img.(*image.NRGBA) // fallback
	}
	log.Printf("reducePalette: successfully reduced to %d colors", n)
	return reduced.(*image.NRGBA)
}

func isWood(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	rf, gf, bf := float64(r)/65535, float64(g)/65535, float64(b)/65535
	h, s, v := rgbToHSV(rf, gf, bf)
	return h >= 15 && h <= 50 && s >= 0.2 && v >= 0.2
}

func rgbToHSV(r, g, b float64) (h, s, v float64) {
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	d := mx - mn

	v = mx
	if mx != 0 {
		s = d / mx
	} else {
		s = 0
		h = -1
		return
	}

	switch mx {
	case r:
		h = (g - b) / d
	case g:
		h = 2 + (b-r)/d
	case b:
		h = 4 + (r-g)/d
	}

	h *= 60
	if h < 0 {
		h += 360
	}

	return
}
