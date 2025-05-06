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
	const outputSize = 256
	outDir := "output"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("cannot create output dir: %v", err)
	}

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

			quad2 := shrinkQuadAligned(quad)

			cropped, err := CropAndCorrect(img, quad2, outputSize)
			if err != nil {
				t.Fatalf("CropAndCorrect: %v", err)
			}

			if cropped.Bounds().Dx() != outputSize || cropped.Bounds().Dy() != outputSize {
				t.Errorf("cropped image size = %dx%d, want %dx%d",
					cropped.Bounds().Dx(), cropped.Bounds().Dy(), outputSize, outputSize)
			}

			base := strings.TrimSuffix(filepath.Base(ti.path), filepath.Ext(ti.path))
			outPath := filepath.Join(outDir, base+"_cropped.png")
			f, err := os.Create(outPath)
			if err != nil {
				t.Fatalf("Create(%s): %v", outPath, err)
			}
			defer f.Close()

			if err := png.Encode(f, cropped); err != nil {
				t.Fatalf("PNG encode failed: %v", err)
			}

			t.Logf("Wrote: %s", outPath)
		})
	}
}

func distance(a, b Point) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}
