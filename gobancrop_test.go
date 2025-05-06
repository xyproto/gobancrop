package gobancrop

import (
	"math"
	"testing"

	"github.com/xyproto/carveimg"
)

var testImages = []struct {
	path string
}{
	{"img/kgs_screenshot1.png"},
	{"img/kgs_screenshot2.png"},
	{"img/kgs_screenshot3.png"},
	{"img/kgs_screenshot4.png"},
	{"img/panda_screenshot1.png"},
}

func TestFindGoban(t *testing.T) {
	for _, ti := range testImages {
		img, err := carveimg.LoadImage(ti.path)
		if err != nil {
			t.Errorf("LoadImage %s: %v", ti.path, err)
			continue
		}
		quad, err := FindGoban(img)
		if err != nil {
			t.Errorf("FindGoban %s returned error: %v", ti.path, err)
			continue
		}
		b := img.Bounds()
		for i, p := range quad {
			if p.X < float64(b.Min.X) || p.X > float64(b.Max.X) ||
				p.Y < float64(b.Min.Y) || p.Y > float64(b.Max.Y) {
				t.Errorf("%s: Quad[%d]=%v out of bounds %v", ti.path, i, p, b)
			}
		}
	}
}

func TestFindActualBoard(t *testing.T) {
	for _, ti := range testImages {
		img, err := carveimg.LoadImage(ti.path)
		if err != nil {
			t.Errorf("LoadImage %s: %v", ti.path, err)
			continue
		}
		quad, err := FindGoban(img)
		if err != nil {
			t.Errorf("FindGoban %s: %v", ti.path, err)
			continue
		}
		quad2, err := FindActualBoard(img, quad)
		if err != nil {
			t.Logf("FindActualBoard %s returned error: %v", ti.path, err)
			continue
		}
		// Check approximate square shape
		d1 := hypot(quad2[0], quad2[1])
		d2 := hypot(quad2[1], quad2[2])
		ratio := d1 / d2
		if ratio < 0.8 || ratio > 1.25 {
			t.Errorf("%s: board quad not square: d1=%.1f d2=%.1f", ti.path, d1, d2)
		}
	}
}

func TestCropAndCorrect(t *testing.T) {
	size := 256
	for _, ti := range testImages {
		img, err := carveimg.LoadImage(ti.path)
		if err != nil {
			t.Errorf("LoadImage %s: %v", ti.path, err)
			continue
		}
		quad, err := FindGoban(img)
		if err != nil {
			t.Errorf("FindGoban %s: %v", ti.path, err)
			continue
		}
		quad2, err := FindActualBoard(img, quad)
		if err != nil {
			t.Logf("Skip CropAndCorrect: %v", err)
			continue
		}
		out, err := CropAndCorrect(img, quad2, size)
		if err != nil {
			t.Errorf("CropAndCorrect %s: %v", ti.path, err)
			continue
		}
		if out.Bounds().Dx() != size || out.Bounds().Dy() != size {
			t.Errorf("%s: cropped size = %dx%d, want %dx%d", ti.path, out.Bounds().Dx(), out.Bounds().Dy(), size, size)
		}
	}
}

func hypot(a, b Point) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}
