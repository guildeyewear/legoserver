package main

import (
	"fmt"
	"log"
	"math"
)

type (
	Point       [2]float64
	Line        [2]Point
	BSpline     []Point
	CubicBezier [4]Point
)

func (p Point) String() string {
	return fmt.Sprintf("{ x: %v, y: %v }", p[0], p[1])
}

// Return the point at a fraction between
// the first and second points defining this line.
// frac should be between 0 and 1
func (l Line) pointAt(frac float64) Point {
	if frac <= 0 {
		return l[0]
	} else if frac >= 1 {
		return l[1]
	}
	return Point{l[0][0] + frac*(l[1][0]-l[0][0]), l[0][1] + frac*(l[1][1]-l[0][1])}
}

func (s Line) String() string {
	return fmt.Sprintf("LINE %v ----> %v", s[0], s[1])
}

// BSplines
func (bs BSpline) minValues() (minx, miny float64) {
	minx = math.MaxFloat64
	miny = math.MaxFloat64
	for _, pt := range bs {
		if pt[0] < minx {
			minx = pt[0]
		}
		if pt[1] < miny {
			miny = pt[1]
		}
	}
	return
}

func (bs BSpline) scale(factor float64) (scaled BSpline) {
	scaled = make(BSpline, len(bs))
	for i, pt := range bs {
		scaled[i] = Point{pt[0] * factor, pt[1] * factor}
	}
	return
}

func (bs BSpline) convertToBeziers(closeCurve, forceHorizontal bool) (beziers []CubicBezier) {
	if closeCurve {
		beziers = make([]CubicBezier, len(bs))
	} else {
		beziers = make([]CubicBezier, len(bs)-1)
	}

	for i := 0; i < len(beziers)-1; i++ {
		// Calculate the two control points of each bezier
		l := Line{bs[i], bs[i+1]}
		beziers[i][1] = l.pointAt(0.333333333)
		beziers[i][2] = l.pointAt(0.666666666)
	}
	// Last point is handled differently for open or closed curves
	if closeCurve {
		l := Line{bs[len(bs)-1], bs[0]}
		beziers[len(beziers)-1][1] = l.pointAt(0.3333333333)
		beziers[len(beziers)-1][2] = l.pointAt(0.6666666666)
	} else {
		idx := len(bs) - 1
		l := Line{bs[idx-1], bs[idx]}
		beziers[len(beziers)-1][1] = l.pointAt(0.333333333)
		beziers[len(beziers)-1][2] = l.pointAt(0.666666666)
	}
	if forceHorizontal {
		// Force open curve to have  horizontal ends
		beziers[0][1][1] = bs[0][1]
		beziers[len(beziers)-1][2][1] = bs[len(bs)-1][1]
	}
	// Calculate the end points of the beziers.  They're at the midpoint between the
	// opposing control points on either side of this bezier.
	// First and last control points are different if the curve is closed, so do those
	// seperately.
	if !closeCurve {
		beziers[0][0] = bs[0]
		beziers[len(beziers)-1][3] = bs[len(bs)-1]
	} else {
		l := Line{beziers[len(beziers)-1][2], beziers[0][1]}
		beziers[0][0] = l.pointAt(0.5)
		beziers[len(beziers)-1][3] = beziers[0][0]
	}
	for i := 0; i < len(beziers)-1; i++ {
		l := Line{beziers[i][2], beziers[i+1][1]}
		beziers[i][3] = l.pointAt(0.5)
		beziers[i+1][0] = beziers[i][3]
	}
	if closeCurve {
		log.Println(beziers)
	}

	return beziers
}
