package generator

import (
	"errors"
	"math"

	"github.com/tdewolff/canvas"
)

// VectorConfig specifies vectorization, scaling, offset and smoothing parameters.
type VectorConfig struct {
	TargetSizeMM     float64 // Maximum dimension in mm (e.g. 70.0)
	BladeThicknessMM float64 // Thickness of cutting blade (e.g. 0.8)
	BaseThicknessMM  float64 // Thickness of handle base flange (e.g. 3.2)
	SimplifyTolMM    float64 // Simplification tolerance in mm (e.g. 0.4)
	SmoothingMM      float64 // Smoothing / rounding strength (e.g. 0.1 to 1.5, default 0.5)
}

// DefaultVectorConfig returns standard default vector configuration.
func DefaultVectorConfig() VectorConfig {
	return VectorConfig{
		TargetSizeMM:     70.0,
		BladeThicknessMM: 0.8,
		BaseThicknessMM:  3.2,
		SimplifyTolMM:    0.4,
		SmoothingMM:      0.5,
	}
}

// BridgeSegment represents a horizontal structural support beam at Z=0..baseHeightMM.
type BridgeSegment struct {
	P1, P2 Point2D
	Width  float64
}

// HoleContour holds the geometry for an interior hole (e.g. the cutout inside letter 'A').
type HoleContour struct {
	OrigRing    []Point2D
	BladeRing   []Point2D
	BaseRing    []Point2D
	IsSolidBase bool // true if base fills the inside of the hole
	Centroid    Point2D
	Bridges     []BridgeSegment

	OrigPath  *canvas.Path
	BladePath *canvas.Path
	BasePath  *canvas.Path
}

// VectorContours holds the primary outer contour and all interior hole contours.
type VectorContours struct {
	OrigRing  []Point2D
	BladeRing []Point2D
	BaseRing  []Point2D

	OrigPath  *canvas.Path
	BladePath *canvas.Path
	BasePath  *canvas.Path

	Holes []*HoleContour
}

// Segment2D represents a directed line segment in pixel coordinates.
type Segment2D struct {
	P1, P2 Point2D
}

type gridKey struct {
	X, Y int
}

func coordToKey(p Point2D) gridKey {
	return gridKey{
		X: int(math.Round(p.X * 100)),
		Y: int(math.Round(p.Y * 100)),
	}
}

// ExtractVectorContours extracts the principal contour and all interior holes from a binary grid,
// applies smoothing to eliminate pixel stepping, scales to TargetSizeMM, and computes offsets and bridges.
func ExtractVectorContours(grid *BinaryGrid, cfg VectorConfig) (*VectorContours, error) {
	if grid == nil || grid.Width < 2 || grid.Height < 2 {
		return nil, errors.New("invalid or too small binary grid")
	}
	if cfg.TargetSizeMM <= 0 {
		cfg.TargetSizeMM = 70.0
	}
	if cfg.BladeThicknessMM <= 0 {
		cfg.BladeThicknessMM = 0.8
	}
	if cfg.BaseThicknessMM <= cfg.BladeThicknessMM {
		cfg.BaseThicknessMM = cfg.BladeThicknessMM + 2.4
	}
	if cfg.SimplifyTolMM <= 0 {
		cfg.SimplifyTolMM = 0.4
	}
	if cfg.SmoothingMM <= 0 {
		cfg.SmoothingMM = 0.5
	}

	// 1. Pad grid with 1 cell border so boundaries are cleanly closed
	paddedW := grid.Width + 2
	paddedH := grid.Height + 2
	padded := make([][]bool, paddedH)
	for y := 0; y < paddedH; y++ {
		padded[y] = make([]bool, paddedW)
	}
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			padded[y+1][x+1] = grid.Data[y][x]
		}
	}

	// 2. Marching Squares line extraction
	segments := marchingSquares(padded, paddedW, paddedH)
	if len(segments) == 0 {
		return nil, errors.New("no contours detected in image")
	}

	// 3. Assemble closed polygon loops
	polygons := assembleLoops(segments)
	if len(polygons) == 0 {
		return nil, errors.New("failed to close contour polygons")
	}

	// Convert coordinates from padded grid to standard cartesian (Y up)
	cartPolygons := make([][]Point2D, len(polygons))
	for i, poly := range polygons {
		cp := make([]Point2D, len(poly))
		for j, pt := range poly {
			cp[j] = Point2D{
				X: pt.X - 1.0,
				Y: float64(grid.Height) - (pt.Y - 1.0),
			}
		}
		cartPolygons[i] = cp
	}

	// 4. Find the principal outer contour (largest absolute area)
	bestIdx := -1
	bestArea := 0.0
	for i, poly := range cartPolygons {
		absArea := math.Abs(polygonArea(poly))
		if absArea > bestArea {
			bestArea = absArea
			bestIdx = i
		}
	}

	if bestIdx == -1 || bestArea < 10.0 {
		return nil, errors.New("no contour with significant area found")
	}

	bestPoly := ensureCCW(cartPolygons[bestIdx])

	// 5. Identify interior hole loops
	var holePolys [][]Point2D
	for i, poly := range cartPolygons {
		if i == bestIdx {
			continue
		}
		absArea := math.Abs(polygonArea(poly))
		// Ignore tiny pixel noise (< 15 px)
		if absArea < 15.0 {
			continue
		}

		// A hole is inside bestPoly
		centroid := polygonCentroid(poly)
		if pointInPolygon(centroid, bestPoly) {
			holePolys = append(holePolys, poly)
		}
	}

	// Calculate smoothing passes from cfg.SmoothingMM
	smoothPasses := int(math.Round(cfg.SmoothingMM * 4.0))
	if smoothPasses < 1 && cfg.SmoothingMM > 0.05 {
		smoothPasses = 1
	}
	if smoothPasses > 6 {
		smoothPasses = 6
	}

	// 6. Smooth, scale and center outer contour
	smoothedOuter := smoothLoop(bestPoly, smoothPasses)
	smoothedOuter = ensureCCW(smoothedOuter)

	origPath := &canvas.Path{}
	origPath.MoveTo(smoothedOuter[0].X, smoothedOuter[0].Y)
	for i := 1; i < len(smoothedOuter); i++ {
		origPath.LineTo(smoothedOuter[i].X, smoothedOuter[i].Y)
	}
	origPath.Close()

	bounds := origPath.Bounds()
	maxDim := math.Max(bounds.W(), bounds.H())
	if maxDim <= 0 {
		return nil, errors.New("contour dimensions are zero")
	}

	scale := cfg.TargetSizeMM / maxDim
	cx := (bounds.X0 + bounds.X1) / 2.0
	cy := (bounds.Y0 + bounds.Y1) / 2.0

	origPath = origPath.Scale(scale, scale)
	origPath = origPath.Translate(-cx*scale, -cy*scale)

	// Simplify path to remove collinear noise and straighten straight edges
	origPath = origPath.SimplifyVisvalingamWhyatt(cfg.SimplifyTolMM)
	if !origPath.CCW() {
		origPath = origPath.Reverse()
	}

	// Generate outer offsets
	bladePath := generateOffset(origPath, cfg.BladeThicknessMM)
	basePath := generateOffset(origPath, cfg.BaseThicknessMM)

	origRing := pathToRing(origPath)
	bladeRing := pathToRing(bladePath)
	baseRing := pathToRing(basePath)

	if len(origRing) < 3 || len(bladeRing) < 3 || len(baseRing) < 3 {
		return nil, errors.New("failed to extract vertex rings from generated contours")
	}

	// 7. Process each interior hole
	var holes []*HoleContour
	bridgeWidth := math.Min(cfg.BaseThicknessMM, 3.0)
	if bridgeWidth < 1.5 {
		bridgeWidth = 1.5
	}

	for _, hPoly := range holePolys {
		sHole := smoothLoop(hPoly, smoothPasses)
		sHole = ensureCCW(sHole)

		hPath := &canvas.Path{}
		hPath.MoveTo(sHole[0].X, sHole[0].Y)
		for i := 1; i < len(sHole); i++ {
			hPath.LineTo(sHole[i].X, sHole[i].Y)
		}
		hPath.Close()

		hPath = hPath.Scale(scale, scale)
		hPath = hPath.Translate(-cx*scale, -cy*scale)
		hPath = hPath.SimplifyVisvalingamWhyatt(cfg.SimplifyTolMM)
		if !hPath.CCW() {
			hPath = hPath.Reverse()
		}

		hOrigRing := pathToRing(hPath)
		if len(hOrigRing) < 3 {
			continue
		}

		// For hole: cutting blade expands inward (-bladeThicknessMM)
		hBladePath := generateInwardOffset(hPath, -cfg.BladeThicknessMM)
		var hBladeRing []Point2D
		if hBladePath != nil && !hBladePath.Empty() {
			hBladeRing = pathToRing(hBladePath)
		}

		// If inward blade is too narrow to offset, fallback to filling or half thickness
		if len(hBladeRing) < 3 {
			hBladePath = generateInwardOffset(hPath, -cfg.BladeThicknessMM*0.5)
			if hBladePath != nil && !hBladePath.Empty() {
				hBladeRing = pathToRing(hBladePath)
			}
		}

		if len(hBladeRing) < 3 {
			// Hole is too tiny for a blade wall, skip
			continue
		}

		// For hole: base expands inward (-baseThicknessMM)
		hBasePath := generateInwardOffset(hPath, -cfg.BaseThicknessMM)
		var hBaseRing []Point2D
		isSolidBase := false
		if hBasePath != nil && !hBasePath.Empty() {
			hBaseRing = pathToRing(hBasePath)
		}
		if len(hBaseRing) < 3 {
			// Base fills the inside of the hole at Z <= baseHeightMM
			isSolidBase = true
			hBaseRing = nil
		}

		hCentroid := polygonCentroid(hOrigRing)

		// 8. Calculate connecting bridges for the hole
		bridges := computeBridges(hOrigRing, origRing, hCentroid, origPath, hPath, bridgeWidth)

		holes = append(holes, &HoleContour{
			OrigRing:    ensureCCW(hOrigRing),
			BladeRing:   ensureCCW(hBladeRing),
			BaseRing:    ensureCCW(hBaseRing),
			IsSolidBase: isSolidBase,
			Centroid:    hCentroid,
			Bridges:     bridges,
			OrigPath:    hPath,
			BladePath:   hBladePath,
			BasePath:    hBasePath,
		})
	}

	return &VectorContours{
		OrigRing:  ensureCCW(origRing),
		BladeRing: ensureCCW(bladeRing),
		BaseRing:  ensureCCW(baseRing),
		OrigPath:  origPath,
		BladePath: bladePath,
		BasePath:  basePath,
		Holes:     holes,
	}, nil
}

// computeBridges finds support beam segments between the hole and the outer frame.
func computeBridges(holeRing, outerRing []Point2D, hc Point2D, outerPath, holePath *canvas.Path, width float64) []BridgeSegment {
	var bridges []BridgeSegment

	// Strategy A: Horizontal ray through centroid
	hLine := canvas.Line(300, 0).Translate(-150, hc.Y)
	outerIntersects := outerPath.Intersections(hLine)
	holeIntersects := holePath.Intersections(hLine)

	var outerLeft, outerRight *canvas.Point
	for i := range outerIntersects {
		pt := &outerIntersects[i]
		if pt.X < hc.X {
			if outerLeft == nil || pt.X > outerLeft.X {
				outerLeft = pt
			}
		} else if pt.X > hc.X {
			if outerRight == nil || pt.X < outerRight.X {
				outerRight = pt
			}
		}
	}

	var holeLeft, holeRight *canvas.Point
	for i := range holeIntersects {
		pt := &holeIntersects[i]
		if pt.X < hc.X {
			if holeLeft == nil || pt.X < holeLeft.X {
				holeLeft = pt
			}
		} else if pt.X > hc.X {
			if holeRight == nil || pt.X > holeRight.X {
				holeRight = pt
			}
		}
	}

	if outerLeft != nil && outerRight != nil && holeLeft != nil && holeRight != nil {
		bridges = append(bridges, BridgeSegment{
			P1:    Point2D{outerLeft.X, hc.Y},
			P2:    Point2D{holeLeft.X, hc.Y},
			Width: width,
		})
		bridges = append(bridges, BridgeSegment{
			P1:    Point2D{holeRight.X, hc.Y},
			P2:    Point2D{outerRight.X, hc.Y},
			Width: width,
		})
		return bridges
	}

	// Strategy B: Shortest distance between holeRing and outerRing
	minDist := 1e9
	var bestPHole, bestPOuter Point2D
	for _, hp := range holeRing {
		for _, op := range outerRing {
			dx := hp.X - op.X
			dy := hp.Y - op.Y
			d := dx*dx + dy*dy
			if d < minDist {
				minDist = d
				bestPHole = hp
				bestPOuter = op
			}
		}
	}

	if minDist < 1e8 {
		bridges = append(bridges, BridgeSegment{
			P1:    bestPHole,
			P2:    bestPOuter,
			Width: width,
		})
	}

	return bridges
}

// generateOffset computes outward offset and selects the outermost contour.
func generateOffset(p *canvas.Path, offsetMM float64) *canvas.Path {
	offsetPath := p.Offset(offsetMM, 0.02)
	offsetPath = offsetPath.Flatten(0.02)

	subpaths := offsetPath.Split()
	if len(subpaths) == 0 {
		return offsetPath
	}
	if len(subpaths) == 1 {
		sp := subpaths[0]
		if !sp.CCW() {
			sp = sp.Reverse()
		}
		return sp
	}

	bestIdx := 0
	bestArea := 0.0
	for i, sp := range subpaths {
		b := sp.Bounds()
		area := b.W() * b.H()
		if area > bestArea {
			bestArea = area
			bestIdx = i
		}
	}
	selected := subpaths[bestIdx]
	if !selected.CCW() {
		selected = selected.Reverse()
	}
	return selected
}

// generateInwardOffset computes inward offset for holes.
func generateInwardOffset(p *canvas.Path, offsetMM float64) *canvas.Path {
	offsetPath := p.Offset(offsetMM, 0.02)
	if offsetPath == nil || offsetPath.Empty() {
		return nil
	}
	offsetPath = offsetPath.Flatten(0.02)

	subpaths := offsetPath.Split()
	if len(subpaths) == 0 {
		return nil
	}
	if len(subpaths) == 1 {
		sp := subpaths[0]
		if !sp.CCW() {
			sp = sp.Reverse()
		}
		return sp
	}

	bestIdx := 0
	bestArea := 0.0
	for i, sp := range subpaths {
		b := sp.Bounds()
		area := b.W() * b.H()
		if area > bestArea {
			bestArea = area
			bestIdx = i
		}
	}
	selected := subpaths[bestIdx]
	if !selected.CCW() {
		selected = selected.Reverse()
	}
	return selected
}

// pathToRing flattens a canvas path and converts it to a slice of unique 2D points.
func pathToRing(p *canvas.Path) []Point2D {
	flat := p.Flatten(0.02)
	coords := flat.Coords()
	if len(coords) < 3 {
		return nil
	}
	n := len(coords)
	if coords[0] == coords[n-1] {
		coords = coords[:n-1]
	}
	pts := make([]Point2D, len(coords))
	for i, c := range coords {
		pts[i] = Point2D{X: c.X, Y: c.Y}
	}
	return pts
}

// marchingSquares extracts boundary segments between true (solid) and false (empty) cells.
func marchingSquares(grid [][]bool, w, h int) []Segment2D {
	var segs []Segment2D

	for y := 0; y < h-1; y++ {
		for x := 0; x < w-1; x++ {
			tl := grid[y][x]
			tr := grid[y][x+1]
			br := grid[y+1][x+1]
			bl := grid[y+1][x]

			cfg := 0
			if tl {
				cfg |= 8
			}
			if tr {
				cfg |= 4
			}
			if br {
				cfg |= 2
			}
			if bl {
				cfg |= 1
			}

			if cfg == 0 || cfg == 15 {
				continue
			}

			fx := float64(x)
			fy := float64(y)
			top := Point2D{fx + 0.5, fy}
			right := Point2D{fx + 1.0, fy + 0.5}
			bottom := Point2D{fx + 0.5, fy + 1.0}
			left := Point2D{fx, fy + 0.5}

			switch cfg {
			case 1: // BL
				segs = append(segs, Segment2D{left, bottom})
			case 2: // BR
				segs = append(segs, Segment2D{bottom, right})
			case 3: // BL + BR
				segs = append(segs, Segment2D{left, right})
			case 4: // TR
				segs = append(segs, Segment2D{right, top})
			case 5: // TR + BL (saddle)
				segs = append(segs, Segment2D{left, top}, Segment2D{right, bottom})
			case 6: // TR + BR
				segs = append(segs, Segment2D{bottom, top})
			case 7: // TR + BR + BL
				segs = append(segs, Segment2D{left, top})
			case 8: // TL
				segs = append(segs, Segment2D{top, left})
			case 9: // TL + BL
				segs = append(segs, Segment2D{top, bottom})
			case 10: // TL + BR (saddle)
				segs = append(segs, Segment2D{top, right}, Segment2D{bottom, left})
			case 11: // TL + BR + BL
				segs = append(segs, Segment2D{top, right})
			case 12: // TL + TR
				segs = append(segs, Segment2D{right, left})
			case 13: // TL + TR + BL
				segs = append(segs, Segment2D{right, bottom})
			case 14: // TL + TR + BR
				segs = append(segs, Segment2D{bottom, left})
			}
		}
	}

	return segs
}

// assembleLoops connects directed segments into closed polygon loops.
func assembleLoops(segs []Segment2D) [][]Point2D {
	nextMap := make(map[gridKey][]int)
	for i, s := range segs {
		k := coordToKey(s.P1)
		nextMap[k] = append(nextMap[k], i)
	}

	used := make([]bool, len(segs))
	var loops [][]Point2D

	for i := 0; i < len(segs); i++ {
		if used[i] {
			continue
		}

		var poly []Point2D
		currIdx := i
		used[currIdx] = true
		poly = append(poly, segs[currIdx].P1)

		for {
			endPt := segs[currIdx].P2
			poly = append(poly, endPt)

			endKey := coordToKey(endPt)
			candidates := nextMap[endKey]
			found := -1
			for _, c := range candidates {
				if !used[c] {
					found = c
					break
				}
			}

			if found == -1 {
				break
			}
			currIdx = found
			used[currIdx] = true

			// Closed loop reached
			if coordToKey(endPt) == coordToKey(poly[0]) {
				break
			}
		}

		if len(poly) >= 4 {
			loops = append(loops, poly)
		}
	}

	return loops
}
