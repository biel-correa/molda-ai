package generator

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strings"
	"testing"
)

// createTestCirclePNG creates an in-memory PNG image of a black circle on white background.
func createTestCirclePNG(w, h int, cx, cy, radius float64) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fill with white
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Draw black circle
	rSq := radius * radius
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= rSq {
				img.Set(x, y, color.Black)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestProcessImage(t *testing.T) {
	pngData := createTestCirclePNG(100, 100, 50, 50, 30)
	cfg := DefaultImageConfig()

	grid, err := ProcessImage(bytes.NewReader(pngData), cfg)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	if grid.Width != 100 || grid.Height != 100 {
		t.Errorf("unexpected dimensions: %dx%d (expected 100x100)", grid.Width, grid.Height)
	}

	// Center should be solid (black pixel)
	if !grid.Data[50][50] {
		t.Errorf("center pixel (50,50) should be solid")
	}

	// Corner should be background (white pixel)
	if grid.Data[5][5] {
		t.Errorf("corner pixel (5,5) should be background")
	}
}

func TestExtractVectorContours(t *testing.T) {
	pngData := createTestCirclePNG(100, 100, 50, 50, 30)
	grid, err := ProcessImage(bytes.NewReader(pngData), DefaultImageConfig())
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	vecCfg := VectorConfig{
		TargetSizeMM:     60.0,
		BladeThicknessMM: 0.8,
		BaseThicknessMM:  3.0,
		SimplifyTolMM:    0.15,
	}

	contours, err := ExtractVectorContours(grid, vecCfg)
	if err != nil {
		t.Fatalf("ExtractVectorContours failed: %v", err)
	}

	if len(contours.OrigRing) < 10 {
		t.Errorf("OrigRing has too few points: %d", len(contours.OrigRing))
	}
	if len(contours.BladeRing) < 10 {
		t.Errorf("BladeRing has too few points: %d", len(contours.BladeRing))
	}
	if len(contours.BaseRing) < 10 {
		t.Errorf("BaseRing has too few points: %d", len(contours.BaseRing))
	}

	// Verify area hierarchy: Base > Blade > Orig
	areaOrig := math.Abs(polygonArea(contours.OrigRing))
	areaBlade := math.Abs(polygonArea(contours.BladeRing))
	areaBase := math.Abs(polygonArea(contours.BaseRing))

	if areaBlade <= areaOrig {
		t.Errorf("blade area (%.1f) must be larger than original (%.1f)", areaBlade, areaOrig)
	}
	if areaBase <= areaBlade {
		t.Errorf("base area (%.1f) must be larger than blade (%.1f)", areaBase, areaBlade)
	}
}

func TestGenerateSTLComplete(t *testing.T) {
	pngData := createTestCirclePNG(120, 120, 60, 60, 40)
	params := CutterParams{
		SizeMM:           70.0,
		BladeThicknessMM: 0.8,
		BladeHeightMM:    12.0,
		BaseThicknessMM:  3.2,
		BaseHeightMM:     3.0,
		Threshold:        128,
		Invert:           false,
	}

	stlBytes, err := GenerateSTL(bytes.NewReader(pngData), params)
	if err != nil {
		t.Fatalf("GenerateSTL failed: %v", err)
	}

	if len(stlBytes) < 84 {
		t.Fatalf("STL size too small: %d bytes", len(stlBytes))
	}

	// Verify binary STL header and facet count
	header := string(stlBytes[:20])
	if header != "MoldaAi Binary STL -" {
		t.Errorf("unexpected STL header: %q", header)
	}

	numFacets := binary.LittleEndian.Uint32(stlBytes[80:84])
	if numFacets == 0 {
		t.Errorf("facet count cannot be zero")
	}

	expectedLen := 84 + int(numFacets)*50
	if len(stlBytes) != expectedLen {
		t.Errorf("invalid STL file length: got %d, expected %d for %d facets", len(stlBytes), expectedLen, numFacets)
	}
}

func TestMeshWatertightManifold(t *testing.T) {
	pngData := createTestCirclePNG(120, 120, 60, 60, 40)
	params := CutterParams{
		SizeMM:           60.0,
		BladeThicknessMM: 0.8,
		BladeHeightMM:    12.0,
		BaseThicknessMM:  3.2,
		BaseHeightMM:     3.0,
		Threshold:        128,
		Invert:           false,
	}

	stlBytes, err := GenerateSTL(bytes.NewReader(pngData), params)
	if err != nil {
		t.Fatalf("GenerateSTL failed: %v", err)
	}

	numFacets := binary.LittleEndian.Uint32(stlBytes[80:84])
	type edgeKey struct {
		x1, y1, z1 int
		x2, y2, z2 int
	}

	roundCoord := func(v float32) int {
		return int(math.Round(float64(v) * 100))
	}

	directedEdges := make(map[edgeKey]int)
	offset := 84
	for i := uint32(0); i < numFacets; i++ {
		// normal (12 bytes)
		offset += 12

		// v1, v2, v3
		v1x := math.Float32frombits(binary.LittleEndian.Uint32(stlBytes[offset : offset+4]))
		v1y := math.Float32frombits(binary.LittleEndian.Uint32(stlBytes[offset+4 : offset+8]))
		v1z := math.Float32frombits(binary.LittleEndian.Uint32(stlBytes[offset+8 : offset+12]))
		offset += 12

		v2x := math.Float32frombits(binary.LittleEndian.Uint32(stlBytes[offset : offset+4]))
		v2y := math.Float32frombits(binary.LittleEndian.Uint32(stlBytes[offset+4 : offset+8]))
		v2z := math.Float32frombits(binary.LittleEndian.Uint32(stlBytes[offset+8 : offset+12]))
		offset += 12

		v3x := math.Float32frombits(binary.LittleEndian.Uint32(stlBytes[offset : offset+4]))
		v3y := math.Float32frombits(binary.LittleEndian.Uint32(stlBytes[offset+4 : offset+8]))
		v3z := math.Float32frombits(binary.LittleEndian.Uint32(stlBytes[offset+8 : offset+12]))
		offset += 12

		// attr (2 bytes)
		offset += 2

		p1 := [3]int{roundCoord(v1x), roundCoord(v1y), roundCoord(v1z)}
		p2 := [3]int{roundCoord(v2x), roundCoord(v2y), roundCoord(v2z)}
		p3 := [3]int{roundCoord(v3x), roundCoord(v3y), roundCoord(v3z)}

		directedEdges[edgeKey{p1[0], p1[1], p1[2], p2[0], p2[1], p2[2]}]++
		directedEdges[edgeKey{p2[0], p2[1], p2[2], p3[0], p3[1], p3[2]}]++
		directedEdges[edgeKey{p3[0], p3[1], p3[2], p1[0], p1[1], p1[2]}]++
	}

	unmatched := 0
	for e, count := range directedEdges {
		opp := edgeKey{e.x2, e.y2, e.z2, e.x1, e.y1, e.z1}
		oppCount := directedEdges[opp]
		if count != oppCount {
			unmatched++
		}
	}

	if unmatched > 0 {
		t.Errorf("STL mesh has %d unmatched edges (not 100%% watertight manifold)", unmatched)
	} else {
		t.Logf("Mesh verified: 100%% watertight manifold, %d facets, 0 open edges!", numFacets)
	}
}

func TestValidationErrors(t *testing.T) {
	pngData := createTestCirclePNG(50, 50, 25, 25, 15)

	// Base height >= blade height must return error
	params := DefaultCutterParams()
	params.BaseHeightMM = 15.0
	params.BladeHeightMM = 10.0

	_, err := GenerateSTL(bytes.NewReader(pngData), params)
	if err == nil {
		t.Errorf("expected error when base_height >= blade_height")
	}
}

func TestLetterAWithHoleAndSmoothing(t *testing.T) {
	imgPath := "/Users/gcorrea/Downloads/3541180.png"
	fileBytes, err := os.ReadFile(imgPath)
	if err != nil {
		t.Skipf("Image %s not found locally, skipping specific test", imgPath)
		return
	}

	params := DefaultCutterParams()
	params.SizeMM = 70.0
	params.SmoothingMM = 0.5

	stlBytes, err := GenerateSTL(bytes.NewReader(fileBytes), params)
	if err != nil {
		t.Fatalf("GenerateSTL failed for letter A: %v", err)
	}

	numFacets := binary.LittleEndian.Uint32(stlBytes[80:84])
	if numFacets < 200 {
		t.Errorf("too few triangles generated for letter A with hole: %d", numFacets)
	}
	t.Logf("Letter A generated successfully! Facets: %d, STL size: %d bytes", numFacets, len(stlBytes))
}

func TestProcessSVGImage(t *testing.T) {
	svgContent := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
	<circle cx="50" cy="50" r="35" fill="black" />
</svg>`

	grid, err := ProcessImage(strings.NewReader(svgContent), DefaultImageConfig())
	if err != nil {
		t.Fatalf("ProcessImage with SVG failed: %v", err)
	}

	if grid.Width <= 0 || grid.Height <= 0 {
		t.Errorf("invalid dimensions for SVG grid: %dx%d", grid.Width, grid.Height)
	}

	params := DefaultCutterParams()
	stlBytes, err := GenerateSTL(strings.NewReader(svgContent), params)
	if err != nil {
		t.Fatalf("GenerateSTL with SVG failed: %v", err)
	}

	numFacets := binary.LittleEndian.Uint32(stlBytes[80:84])
	if numFacets == 0 {
		t.Errorf("expected triangular facets from SVG")
	}
	t.Logf("SVG processed successfully! Facets: %d, STL: %d bytes", numFacets, len(stlBytes))
}
