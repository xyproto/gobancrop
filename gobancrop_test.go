package gobancrop

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xyproto/carveimg"
)

var testImages = []struct {
	name, path string
}{
	{"KGS1", "img/kgs_screenshot1.png"},
	{"KGS2", "img/kgs_screenshot2.png"},
	{"KGS3", "img/kgs_screenshot3.png"},
	{"KGS4", "img/kgs_screenshot4.png"},
	{"Panda1", "img/panda_screenshot1.png"},
}

func TestGobancropPipeline(t *testing.T) {
	outDir := filepath.Join("..", "output")
	if err := os.RemoveAll(outDir); err != nil {
		t.Fatalf("failed to clear output dir: %v", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	size := 256
	for _, ti := range testImages {
		t.Run(ti.name, func(t *testing.T) {
			img, err := carveimg.LoadImage(ti.path)
			if err != nil {
				t.Fatalf("LoadImage(%s): %v", ti.path, err)
			}

			quad, err := FindGoban(img)
			if err != nil {
				t.Fatalf("FindGoban: %v", err)
			}
			b := img.Bounds()
			for i, p := range quad {
				if p.X < float64(b.Min.X) || p.X > float64(b.Max.X) ||
					p.Y < float64(b.Min.Y) || p.Y > float64(b.Max.Y) {
					t.Fatalf("Quad[%d]=%v out of bounds %v", i, p, b)
				}
			}

			quad2, err := FindActualBoard(img, quad)
			if err != nil {
				t.Logf("FindActualBoard failed, using initial quad: %v", err)
				quad2 = quad
			}

			d1 := hypot(quad2[0], quad2[1])
			d2 := hypot(quad2[1], quad2[2])
			ratio := d1 / d2
			if ratio < 0.8 || ratio > 1.25 {
				t.Errorf("board not square: ratio=%.2f (d1=%.1f d2=%.1f)", ratio, d1, d2)
			}

			outImg, err := CropAndCorrect(img, quad2, size)
			if err != nil {
				t.Fatalf("CropAndCorrect: %v", err)
			}
			if w, h := outImg.Bounds().Dx(), outImg.Bounds().Dy(); w != size || h != size {
				t.Fatalf("cropped size = %dx%d, want %dx%d", w, h, size, size)
			}

			base := strings.TrimSuffix(filepath.Base(ti.path), filepath.Ext(ti.path))
			outPath := filepath.Join(outDir, base+"_cropped.png")
			f, err := os.Create(outPath)
			if err != nil {
				t.Fatalf("create file: %v", err)
			}
			if err := png.Encode(f, outImg); err != nil {
				f.Close()
				t.Fatalf("encode PNG: %v", err)
			}
			f.Close()
			// verify file exists
			if _, err := os.Stat(outPath); err != nil {
				t.Errorf("output file missing: %v", err)
			}
		})
	}
}

func hypot(a, b Point) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Hypot(dx, dy)
}
