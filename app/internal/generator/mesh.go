package generator

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/rclancey/earcut"
)

// Vec3 represents a 3D vertex or normal vector in millimeters.
type Vec3 struct {
	X, Y, Z float32
}

// Triangle represents a single facet in the STL mesh.
type Triangle struct {
	Normal     Vec3
	V1, V2, V3 Vec3
}

// MeshConfig defines the heights and dimensions for the 3D model.
type MeshConfig struct {
	BladeHeightMM float64 // Total cutter height (e.g. 12.0)
	BaseHeightMM  float64 // Flange/support height (e.g. 3.0)
}

// DefaultMeshConfig returns the standard mesh height configuration.
func DefaultMeshConfig() MeshConfig {
	return MeshConfig{
		BladeHeightMM: 12.0,
		BaseHeightMM:  3.0,
	}
}

// CutterParams combines all parameters for a generation request.
type CutterParams struct {
	SizeMM           float64
	BladeThicknessMM float64
	BladeHeightMM    float64
	BaseThicknessMM  float64
	BaseHeightMM     float64
	Threshold        int
	Invert           bool
	SmoothingMM      float64
}

// DefaultCutterParams returns the recommended default parameters.
func DefaultCutterParams() CutterParams {
	return CutterParams{
		SizeMM:           70.0,
		BladeThicknessMM: 0.8,
		BladeHeightMM:    12.0,
		BaseThicknessMM:  3.2,
		BaseHeightMM:     3.0,
		Threshold:        128,
		Invert:           false,
		SmoothingMM:      0.5,
	}
}

// BuildMesh creates a watertight 3D cookie/clay cutter mesh from the vector contours,
// including all interior hole cutters and structural base support bridges.
func BuildMesh(contours *VectorContours, cfg MeshConfig) ([]Triangle, error) {
	if contours == nil {
		return nil, errors.New("nil vector contours")
	}
	if len(contours.OrigRing) < 3 || len(contours.BladeRing) < 3 || len(contours.BaseRing) < 3 {
		return nil, errors.New("insufficient contour rings for 3D extrusion")
	}
	if cfg.BladeHeightMM <= 0 {
		cfg.BladeHeightMM = 12.0
	}
	if cfg.BaseHeightMM <= 0 {
		cfg.BaseHeightMM = 3.0
	}
	if cfg.BaseHeightMM >= cfg.BladeHeightMM {
		return nil, fmt.Errorf("base height (%.1f mm) must be less than total blade height (%.1f mm)", cfg.BaseHeightMM, cfg.BladeHeightMM)
	}

	z0 := float32(0.0)
	zBase := float32(cfg.BaseHeightMM)
	zBlade := float32(cfg.BladeHeightMM)

	var triangles []Triangle

	// 1. --- Outer Cutter Shell ---

	// 1.1 Bottom Face (Z = 0): Annulus between Base and Orig.
	botTris, err := triangulateAnnulus(contours.BaseRing, contours.OrigRing, z0, true)
	if err != nil {
		return nil, fmt.Errorf("failed to triangulate bottom base face: %w", err)
	}
	triangles = append(triangles, botTris...)

	// 1.2 Outer Base Wall: Extrusion of BaseRing from Z=0 to Z=BaseHeight
	baseWall := extrudeWall(contours.BaseRing, z0, zBase, false)
	triangles = append(triangles, baseWall...)

	// 1.3 Step/Shelf Face (Z = BaseHeight): Annulus between Base and Blade.
	shelfTris, err := triangulateAnnulus(contours.BaseRing, contours.BladeRing, zBase, false)
	if err != nil {
		return nil, fmt.Errorf("failed to triangulate base shelf support face: %w", err)
	}
	triangles = append(triangles, shelfTris...)

	// 1.4 Outer Blade Wall: Extrusion of BladeRing from Z=BaseHeight to Z=BladeHeight
	bladeWall := extrudeWall(contours.BladeRing, zBase, zBlade, false)
	triangles = append(triangles, bladeWall...)

	// 1.5 Top Blade Cutting Face (Z = BladeHeight): Annulus between Blade and Orig.
	topTris, err := triangulateAnnulus(contours.BladeRing, contours.OrigRing, zBlade, false)
	if err != nil {
		return nil, fmt.Errorf("failed to triangulate cutting blade face: %w", err)
	}
	triangles = append(triangles, topTris...)

	// 1.6 Inner Cavity Wall: Extrusion of OrigRing from Z=0 to Z=BladeHeight.
	innerWall := extrudeWall(contours.OrigRing, z0, zBlade, true)
	triangles = append(triangles, innerWall...)

	// 2. --- Interior Hole Cutters ---
	for hIdx, hole := range contours.Holes {
		if len(hole.OrigRing) < 3 || len(hole.BladeRing) < 3 {
			continue
		}

		// 2.1 Top Cutting Face (Z = BladeHeight): Annulus between OrigRing and BladeRing.
		// For hole: OrigRing is outer cutting boundary, BladeRing is inward hollow.
		hTopTris, err := triangulateAnnulus(hole.OrigRing, hole.BladeRing, zBlade, false)
		if err != nil {
			return nil, fmt.Errorf("failed to triangulate blade of hole %d: %w", hIdx, err)
		}
		triangles = append(triangles, hTopTris...)

		// 2.2 Outer Cutting Wall: Along OrigRing from Z=0 to Z=BladeHeight.
		// Normal points OUTWARD (towards dough).
		hCuttingWall := extrudeWall(hole.OrigRing, z0, zBlade, false)
		triangles = append(triangles, hCuttingWall...)

		// 2.3 Inner Hollow Wall of Blade: Along BladeRing from Z=BaseHeight to Z=BladeHeight.
		// Normal points INWARD (into hollow).
		hBladeInnerWall := extrudeWall(hole.BladeRing, zBase, zBlade, true)
		triangles = append(triangles, hBladeInnerWall...)

		if !hole.IsSolidBase && len(hole.BaseRing) >= 3 {
			// Case A: Hollow base
			// Shelf face at Z = BaseHeight: Annulus between BladeRing and BaseRing.
			hShelfTris, err := triangulateAnnulus(hole.BladeRing, hole.BaseRing, zBase, false)
			if err != nil {
				return nil, fmt.Errorf("failed to triangulate shelf of hole %d: %w", hIdx, err)
			}
			triangles = append(triangles, hShelfTris...)

			// Base inner wall from Z=0 to Z=BaseHeight along BaseRing
			hBaseInnerWall := extrudeWall(hole.BaseRing, z0, zBase, true)
			triangles = append(triangles, hBaseInnerWall...)

			// Bottom face at Z = 0: Annulus between OrigRing and BaseRing. Normal DOWN.
			hBotTris, err := triangulateAnnulus(hole.OrigRing, hole.BaseRing, z0, true)
			if err != nil {
				return nil, fmt.Errorf("failed to triangulate bottom base of hole %d: %w", hIdx, err)
			}
			triangles = append(triangles, hBotTris...)
		} else {
			// Case B: Solid base floor inside hole up to Z = BaseHeight
			// Shelf floor at Z = BaseHeight: Solid cap of BladeRing. Normal UP.
			hShelfFloor, err := triangulateSolidCap(hole.BladeRing, zBase, false)
			if err != nil {
				return nil, fmt.Errorf("failed to triangulate floor of hole %d: %w", hIdx, err)
			}
			triangles = append(triangles, hShelfFloor...)

			// Bottom face at Z = 0: Solid cap of OrigRing. Normal DOWN.
			hBotSolid, err := triangulateSolidCap(hole.OrigRing, z0, true)
			if err != nil {
				return nil, fmt.Errorf("failed to triangulate solid bottom of hole %d: %w", hIdx, err)
			}
			triangles = append(triangles, hBotSolid...)
		}

		// 2.4 Structural Connecting Bridges
		for _, bridge := range hole.Bridges {
			bTris := extrudeBridge(bridge, z0, zBase)
			triangles = append(triangles, bTris...)
		}
	}

	return triangles, nil
}

// extrudeBridge creates a solid 3D structural beam connecting a hole to the outer frame at Z=zMin..zMax.
func extrudeBridge(seg BridgeSegment, zMin, zMax float32) []Triangle {
	dx := seg.P2.X - seg.P1.X
	dy := seg.P2.Y - seg.P1.Y
	l := math.Sqrt(dx*dx + dy*dy)
	if l < 1e-6 {
		return nil
	}

	ux := dx / l
	uy := dy / l
	nx := -uy
	ny := ux

	hw := seg.Width / 2.0
	// Slightly lengthen by 0.5mm at both ends to ensure robust overlap with bases
	p1ExtX := seg.P1.X - ux*0.5
	p1ExtY := seg.P1.Y - uy*0.5
	p2ExtX := seg.P2.X + ux*0.5
	p2ExtY := seg.P2.Y + uy*0.5

	c1 := Point2D{p1ExtX - nx*hw, p1ExtY - ny*hw}
	c2 := Point2D{p2ExtX - nx*hw, p2ExtY - ny*hw}
	c3 := Point2D{p2ExtX + nx*hw, p2ExtY + ny*hw}
	c4 := Point2D{p1ExtX + nx*hw, p1ExtY + ny*hw}

	quadRing := []Point2D{c1, c2, c3, c4}

	var tris []Triangle

	// Top Face (Z = zMax): Normal UP (0, 0, 1)
	topTris, _ := triangulateSolidCap(quadRing, zMax, false)
	tris = append(tris, topTris...)

	// Bottom Face (Z = zMin): Normal DOWN (0, 0, -1)
	botTris, _ := triangulateSolidCap(quadRing, zMin, true)
	tris = append(tris, botTris...)

	// 4 Vertical Side Walls
	wallTris := extrudeWall(quadRing, zMin, zMax, false)
	tris = append(tris, wallTris...)

	return tris
}

// triangulateSolidCap creates triangles for a closed 2D polygon with no holes.
func triangulateSolidCap(ring []Point2D, z float32, invertNormal bool) ([]Triangle, error) {
	pts := ensureCCW(ring)
	data := make([]float64, 0, len(pts)*2)
	for _, p := range pts {
		data = append(data, p.X, p.Y)
	}

	indices, err := earcut.Earcut(data, nil, 2)
	if err != nil {
		return nil, err
	}

	allPts := make([]Vec3, len(pts))
	for i, p := range pts {
		allPts[i] = Vec3{float32(p.X), float32(p.Y), z}
	}

	triangles := make([]Triangle, 0, len(indices)/3)
	for i := 0; i < len(indices); i += 3 {
		v1 := allPts[indices[i]]
		v2 := allPts[indices[i+1]]
		v3 := allPts[indices[i+2]]

		if invertNormal {
			v2, v3 = v3, v2
		}

		n := calculateFaceNormal(v1, v2, v3)
		triangles = append(triangles, Triangle{Normal: n, V1: v1, V2: v2, V3: v3})
	}

	return triangles, nil
}

// triangulateAnnulus creates triangles between an outer ring and an inner hole ring using Earcut.
func triangulateAnnulus(outer []Point2D, inner []Point2D, z float32, invertNormal bool) ([]Triangle, error) {
	outerCCW := ensureCCW(outer)
	innerCW := ensureCW(inner)

	data := make([]float64, 0, (len(outerCCW)+len(innerCW))*2)
	for _, p := range outerCCW {
		data = append(data, p.X, p.Y)
	}
	holeIndex := len(outerCCW)
	for _, p := range innerCW {
		data = append(data, p.X, p.Y)
	}

	indices, err := earcut.Earcut(data, []int{holeIndex}, 2)
	if err != nil {
		return nil, err
	}

	allPts := make([]Vec3, 0, len(outerCCW)+len(innerCW))
	for _, p := range outerCCW {
		allPts = append(allPts, Vec3{float32(p.X), float32(p.Y), z})
	}
	for _, p := range innerCW {
		allPts = append(allPts, Vec3{float32(p.X), float32(p.Y), z})
	}

	triangles := make([]Triangle, 0, len(indices)/3)
	for i := 0; i < len(indices); i += 3 {
		idx1 := indices[i]
		idx2 := indices[i+1]
		idx3 := indices[i+2]

		v1 := allPts[idx1]
		v2 := allPts[idx2]
		v3 := allPts[idx3]

		if invertNormal {
			v2, v3 = v3, v2
		}

		n := calculateFaceNormal(v1, v2, v3)
		triangles = append(triangles, Triangle{Normal: n, V1: v1, V2: v2, V3: v3})
	}

	return triangles, nil
}

// extrudeWall creates vertical wall quads between ring vertices at zMin and zMax.
func extrudeWall(ring []Point2D, zMin, zMax float32, isHole bool) []Triangle {
	var pts []Point2D
	if isHole {
		pts = ensureCW(ring)
	} else {
		pts = ensureCCW(ring)
	}

	n := len(pts)
	triangles := make([]Triangle, 0, n*2)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		p1 := pts[i]
		p2 := pts[j]

		b1 := Vec3{float32(p1.X), float32(p1.Y), zMin}
		b2 := Vec3{float32(p2.X), float32(p2.Y), zMin}
		t1 := Vec3{float32(p1.X), float32(p1.Y), zMax}
		t2 := Vec3{float32(p2.X), float32(p2.Y), zMax}

		tri1 := Triangle{V1: b1, V2: b2, V3: t2}
		tri1.Normal = calculateFaceNormal(tri1.V1, tri1.V2, tri1.V3)

		tri2 := Triangle{V1: b1, V2: t2, V3: t1}
		tri2.Normal = calculateFaceNormal(tri2.V1, tri2.V2, tri2.V3)

		triangles = append(triangles, tri1, tri2)
	}

	return triangles
}

// calculateFaceNormal computes the normalized right-handed surface normal.
func calculateFaceNormal(v1, v2, v3 Vec3) Vec3 {
	ax := v2.X - v1.X
	ay := v2.Y - v1.Y
	az := v2.Z - v1.Z

	bx := v3.X - v1.X
	by := v3.Y - v1.Y
	bz := v3.Z - v1.Z

	nx := ay*bz - az*by
	ny := az*bx - ax*bz
	nz := ax*by - ay*bx

	lenSq := nx*nx + ny*ny + nz*nz
	if lenSq > 1e-12 {
		invLen := 1.0 / float32(math.Sqrt(float64(lenSq)))
		nx *= invLen
		ny *= invLen
		nz *= invLen
	} else {
		nx, ny, nz = 0, 0, 1
	}

	return Vec3{nx, ny, nz}
}

// SerializeSTL writes triangles into the standard Binary STL format.
func SerializeSTL(triangles []Triangle) []byte {
	buf := new(bytes.Buffer)

	header := make([]byte, 80)
	copy(header, []byte("MoldaAi Binary STL - Single-Binary 3D Cutter Generator"))
	buf.Write(header)

	count := uint32(len(triangles))
	binary.Write(buf, binary.LittleEndian, count)

	for _, t := range triangles {
		binary.Write(buf, binary.LittleEndian, t.Normal)
		binary.Write(buf, binary.LittleEndian, t.V1)
		binary.Write(buf, binary.LittleEndian, t.V2)
		binary.Write(buf, binary.LittleEndian, t.V3)
		binary.Write(buf, binary.LittleEndian, uint16(0))
	}

	return buf.Bytes()
}

// GenerateSTL executes the complete pipeline from input image reader to binary STL bytes.
func GenerateSTL(r io.Reader, params CutterParams) ([]byte, error) {
	// Step 1: Image Processing
	imgCfg := ImageConfig{
		Threshold: params.Threshold,
		Invert:    params.Invert,
		MaxPixels: 1000,
	}
	grid, err := ProcessImage(r, imgCfg)
	if err != nil {
		return nil, fmt.Errorf("image processing: %w", err)
	}

	// Step 2: Vectorization, Smoothing & Offset
	simplifyTol := params.SmoothingMM
	if simplifyTol <= 0 {
		simplifyTol = 0.4
	}
	vecCfg := VectorConfig{
		TargetSizeMM:     params.SizeMM,
		BladeThicknessMM: params.BladeThicknessMM,
		BaseThicknessMM:  params.BaseThicknessMM,
		SimplifyTolMM:    simplifyTol,
		SmoothingMM:      params.SmoothingMM,
	}
	contours, err := ExtractVectorContours(grid, vecCfg)
	if err != nil {
		return nil, fmt.Errorf("contour vectorization: %w", err)
	}

	// Step 3: 3D Extrusion (Outer, Holes and Bridges)
	meshCfg := MeshConfig{
		BladeHeightMM: params.BladeHeightMM,
		BaseHeightMM:  params.BaseHeightMM,
	}
	triangles, err := BuildMesh(contours, meshCfg)
	if err != nil {
		return nil, fmt.Errorf("3D mesh generation: %w", err)
	}

	// Step 4: Binary STL Serialization
	stlBytes := SerializeSTL(triangles)
	return stlBytes, nil
}
