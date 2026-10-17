package generator

// Point2D represents a 2D coordinate in millimeters.
type Point2D struct {
	X, Y float64
}

// polygonArea computes the signed 2D area using the Shoelace formula.
// Positive area means CCW orientation, negative means CW.
func polygonArea(pts []Point2D) float64 {
	n := len(pts)
	if n < 3 {
		return 0
	}
	area := 0.0
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
	}
	return area / 2.0
}

// ensureCCW returns points in Counter-Clockwise order (positive area).
func ensureCCW(pts []Point2D) []Point2D {
	if polygonArea(pts) < 0 {
		return reversePoints(pts)
	}
	return pts
}

// ensureCW returns points in Clockwise order (negative area).
func ensureCW(pts []Point2D) []Point2D {
	if polygonArea(pts) > 0 {
		return reversePoints(pts)
	}
	return pts
}

// reversePoints returns a reversed slice of points.
func reversePoints(pts []Point2D) []Point2D {
	n := len(pts)
	rev := make([]Point2D, n)
	for i := 0; i < n; i++ {
		rev[i] = pts[n-1-i]
	}
	return rev
}

// pointInPolygon tests whether a 2D point is inside a polygon using ray casting.
func pointInPolygon(pt Point2D, poly []Point2D) bool {
	inside := false
	n := len(poly)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		xi, yi := poly[i].X, poly[i].Y
		xj, yj := poly[j].X, poly[j].Y

		intersect := ((yi > pt.Y) != (yj > pt.Y)) &&
			(pt.X < (xj-xi)*(pt.Y-yi)/(yj-yi)+xi)
		if intersect {
			inside = !inside
		}
	}
	return inside
}

// polygonCentroid computes the center of mass of the polygon vertices.
func polygonCentroid(poly []Point2D) Point2D {
	if len(poly) == 0 {
		return Point2D{0, 0}
	}
	cx, cy := 0.0, 0.0
	for _, p := range poly {
		cx += p.X
		cy += p.Y
	}
	return Point2D{cx / float64(len(poly)), cy / float64(len(poly))}
}

// smoothLoop applies Laplacian moving average filtering to remove pixel stair-stepping.
func smoothLoop(pts []Point2D, iterations int) []Point2D {
	n := len(pts)
	if n < 4 || iterations <= 0 {
		return pts
	}
	curr := make([]Point2D, n)
	copy(curr, pts)

	for iter := 0; iter < iterations; iter++ {
		next := make([]Point2D, n)
		for i := 0; i < n; i++ {
			prev := (i - 1 + n) % n
			fwd := (i + 1) % n
			next[i] = Point2D{
				X: 0.25*curr[prev].X + 0.5*curr[i].X + 0.25*curr[fwd].X,
				Y: 0.25*curr[prev].Y + 0.5*curr[i].Y + 0.25*curr[fwd].Y,
			}
		}
		curr = next
	}
	return curr
}
