package generator

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/webp"
)

func init() {
	// Register WebP format with image.Decode
	image.RegisterFormat("webp", "RIFF????WEBPVP8", webp.Decode, webp.DecodeConfig)
}

// ImageConfig holds the parameters for image processing.
type ImageConfig struct {
	Threshold int  // 1 to 254 (default 128)
	Invert    bool // if true, flip solid and background
	MaxPixels int  // max allowed size on the larger dimension (e.g. 800-1200)
}

// DefaultImageConfig returns the standard image processing configuration.
func DefaultImageConfig() ImageConfig {
	return ImageConfig{
		Threshold: 128,
		Invert:    false,
		MaxPixels: 1000,
	}
}

// BinaryGrid represents a 2D boolean mask where true means solid material and false means background.
type BinaryGrid struct {
	Width  int
	Height int
	Data   [][]bool
}

// isSVGContent checks if the beginning of data looks like an SVG or XML file.
func isSVGContent(data []byte) bool {
	trim := strings.TrimSpace(string(data))
	if len(trim) > 500 {
		trim = trim[:500]
	}
	lower := strings.ToLower(trim)
	return strings.Contains(lower, "<svg") || (strings.HasPrefix(lower, "<?xml") && strings.Contains(lower, "<svg"))
}

// decodeSVG parses and rasterizes SVG data into an image.Image.
func decodeSVG(data []byte, maxDim int) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SVG vector file: %w", err)
	}

	w := int(icon.ViewBox.W)
	h := int(icon.ViewBox.H)
	if w <= 0 || h <= 0 {
		w, h = 800, 800
	}

	if maxDim > 0 {
		if w > maxDim || h > maxDim {
			if w > h {
				h = (h * maxDim) / w
				w = maxDim
			} else {
				w = (w * maxDim) / h
				h = maxDim
			}
		}
	}
	if w < 10 {
		w = 10
	}
	if h < 10 {
		h = 10
	}

	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	dasher := rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, rgba, rgba.Bounds()))
	icon.Draw(dasher, 1.0)

	return rgba, nil
}

// ProcessImage decodes an image (PNG, JPEG, WebP, SVG, GIF) from reader,
// converts it to grayscale luminance, applies binary thresholding, and returns a BinaryGrid.
func ProcessImage(r io.Reader, cfg ImageConfig) (*BinaryGrid, error) {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 128
	}
	if cfg.MaxPixels <= 0 {
		cfg.MaxPixels = 1000
	}

	// Read all bytes to allow sniffing format and multiple decoding attempts
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("empty image file")
	}

	var img image.Image

	// 1. Check for SVG
	if isSVGContent(data) {
		svgImg, err := decodeSVG(data, cfg.MaxPixels)
		if err == nil {
			img = svgImg
		}
	}

	// 2. Standard image.Decode (PNG, JPEG, GIF, WebP)
	if img == nil {
		decoded, _, err := image.Decode(bytes.NewReader(data))
		if err == nil {
			img = decoded
		}
	}

	// 3. Fallback explicit WebP decode
	if img == nil {
		webpImg, err := webp.Decode(bytes.NewReader(data))
		if err == nil {
			img = webpImg
		}
	}

	if img == nil {
		return nil, errors.New("unsupported image format or corrupted file (supported: PNG, JPEG, WebP, SVG)")
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("image has invalid dimensions")
	}

	// Rescale if image exceeds MaxPixels on larger dimension
	scale := 1.0
	maxDim := math.Max(float64(w), float64(h))
	if maxDim > float64(cfg.MaxPixels) {
		scale = float64(cfg.MaxPixels) / maxDim
	}

	targetW := int(math.Round(float64(w) * scale))
	targetH := int(math.Round(float64(h) * scale))
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}

	gridData := make([][]bool, targetH)
	solidCount := 0

	for y := 0; y < targetH; y++ {
		gridData[y] = make([]bool, targetW)
		origY := bounds.Min.Y + int(math.Floor(float64(y)/scale))
		if origY >= bounds.Max.Y {
			origY = bounds.Max.Y - 1
		}

		for x := 0; x < targetW; x++ {
			origX := bounds.Min.X + int(math.Floor(float64(x)/scale))
			if origX >= bounds.Max.X {
				origX = bounds.Max.X - 1
			}

			c := img.At(origX, origY)
			isSolid := pixelToSolid(c, cfg.Threshold, cfg.Invert)
			gridData[y][x] = isSolid
			if isSolid {
				solidCount++
			}
		}
	}

	if solidCount == 0 {
		return nil, errors.New("binarized image resulted in empty area (no solid pixels detected). Try adjusting threshold or inverting colors")
	}

	if solidCount == targetW*targetH {
		return nil, errors.New("binarized image is 100% solid (no hollow contours detected). Try adjusting threshold or inverting colors")
	}

	return &BinaryGrid{
		Width:  targetW,
		Height: targetH,
		Data:   gridData,
	}, nil
}

// pixelToSolid determines if a color pixel is solid based on luminance and alpha.
func pixelToSolid(c color.Color, threshold int, invert bool) bool {
	r, g, b, a := c.RGBA()
	// RGBA() returns values in [0, 0xffff]

	// Check alpha: if transparent or semi-transparent (a < 50%), treat as background
	if a < 0x8000 {
		if invert {
			return true
		}
		return false
	}

	// Normalize to 0-255 using ITU-R BT.601 luminance
	rf := float64(r >> 8)
	gf := float64(g >> 8)
	bf := float64(b >> 8)
	luminance := int(math.Round(0.299*rf + 0.587*gf + 0.114*bf))

	// By default: dark pixels (< threshold) are solid (cookie cutter silhouette)
	isDark := luminance < threshold

	if invert {
		return !isDark
	}
	return isDark
}
