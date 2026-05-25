// Package imaging provides image utilities: mock generation, resolution
// validation, and cropping helpers used during PDF assembly.
package imaging

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

// WriteMockImage writes a white PNG of the given pixel dimensions to path.
// Two diagonal lines (corner to corner) are drawn in light grey to make the
// placeholder visually distinct from a missing-image error.
func WriteMockImage(path string, widthPx, heightPx int) error {
	img := image.NewRGBA(image.Rect(0, 0, widthPx, heightPx))

	// Fill white.
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	// Draw diagonal cross in light grey.
	grey := color.RGBA{R: 180, G: 180, B: 180, A: 255}
	drawLine(img, 0, 0, widthPx-1, heightPx-1, grey)
	drawLine(img, widthPx-1, 0, 0, heightPx-1, grey)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encoding PNG: %w", err)
	}
	return nil
}

// ValidateResolution checks whether an image file at path has enough pixels
// to print at targetDPI for the given physical dimensions (in inches).
// Returns an error if the image is too small or cannot be read.
func ValidateResolution(path string, widthInches, heightInches float64, targetDPI int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening image: %w", err)
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return fmt.Errorf("decoding image config: %w", err)
	}

	minW := int(widthInches * float64(targetDPI))
	minH := int(heightInches * float64(targetDPI))

	if cfg.Width < minW || cfg.Height < minH {
		return fmt.Errorf(
			"image %q is %dx%d px; minimum for %.3f×%.3f\" at %d DPI is %dx%d px",
			path, cfg.Width, cfg.Height, widthInches, heightInches, targetDPI, minW, minH,
		)
	}
	return nil
}

// drawLine draws a 1-pixel-wide line between (x0,y0) and (x1,y1) using
// Bresenham's algorithm.
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	bounds := img.Bounds()
	for {
		if bounds.Min.X <= x0 && x0 < bounds.Max.X &&
			bounds.Min.Y <= y0 && y0 < bounds.Max.Y {
			img.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}