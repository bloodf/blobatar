// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar vendor/core/shape.ts, MIT License.
// See NOTICE for the full license text.

package blobatar

import (
	"math"
	"strconv"
	"strings"
)

// shapeNumber matches JavaScript Math.round's ties toward positive infinity
// and removes negative zero, as the source SVG path generator does.
func shapeNumber(v float64) string {
	whole, fraction := math.Modf(v * 100)
	if fraction >= 0.5 {
		whole++
	} else if fraction < -0.5 {
		whole--
	}
	n := whole / 100
	if n == 0 {
		return "0"
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}

// superellipse emits the four cubic quadrants used by the pinned source.
func superellipse(cx, cy, rx, ry, n, rot float64) string {
	k := math.Min(1, (float64(8*math.Pow(2, -1/n))-4)/3)
	ak, bk := rx*k, ry*k
	pts := []point{{rx, 0}, {rx, bk}, {ak, ry}, {0, ry}, {-ak, ry}, {-rx, bk}, {-rx, 0}, {-rx, -bk}, {-ak, -ry}, {0, -ry}, {ak, -ry}, {rx, -bk}, {rx, 0}}
	angle := rot * math.Pi / 180
	cos, sin := math.Cos(angle), math.Sin(angle)
	at := func(i int) string {
		p := pts[i]
		return shapeNumber(float64(cx+float64(p[0]*cos))-float64(p[1]*sin)) + " " + shapeNumber(float64(cy+float64(p[0]*sin))+float64(p[1]*cos))
	}
	var b strings.Builder
	b.WriteString("M")
	b.WriteString(at(0))
	for i := 1; i < 13; i += 3 {
		b.WriteString("C")
		b.WriteString(at(i))
		b.WriteByte(' ')
		b.WriteString(at(i + 1))
		b.WriteByte(' ')
		b.WriteString(at(i + 2))
	}
	b.WriteString("Z")
	return b.String()
}

// blobPath creates the source's closed Catmull-Rom silhouette.
func blobPath(cx, cy, rx, ry float64, radii []float64, rot float64) string {
	n := len(radii)
	if n == 0 {
		return ""
	}
	start := rot * math.Pi / 180
	pts := make([]point, n)
	for i, m := range radii {
		a := start + 2*math.Pi*float64(i)/float64(n)
		pts[i] = point{cx + float64(rx*m*math.Cos(a)), cy + float64(ry*m*math.Sin(a))}
	}
	at := func(i int) point { return pts[(i%n+n)%n] }
	pair := func(p point) string { return shapeNumber(p[0]) + " " + shapeNumber(p[1]) }
	var b strings.Builder
	b.WriteString("M")
	b.WriteString(pair(at(0)))
	for i := 0; i < n; i++ {
		p0, p1, p2, p3 := at(i-1), at(i), at(i+1), at(i+2)
		c1 := point{p1[0] + (p2[0]-p0[0])/6, p1[1] + (p2[1]-p0[1])/6}
		c2 := point{p2[0] - (p3[0]-p1[0])/6, p2[1] - (p3[1]-p1[1])/6}
		b.WriteString("C")
		b.WriteString(pair(c1))
		b.WriteByte(' ')
		b.WriteString(pair(c2))
		b.WriteByte(' ')
		b.WriteString(pair(p2))
	}
	b.WriteString("Z")
	return b.String()
}

func polygonPath(cx, cy, rx, ry float64, sides int, round, rot float64) string {
	if sides <= 0 {
		return ""
	}
	k := 0.0
	if round > 0 {
		if round < 1 {
			k = round / 2
		} else {
			k = 0.5
		}
	}
	start := rot*math.Pi/180 - math.Pi/2
	pts := make([]point, sides)
	for i := range pts {
		a := start + 2*math.Pi*float64(i)/float64(sides)
		pts[i] = point{cx + float64(rx*math.Cos(a)), cy + float64(ry*math.Sin(a))}
	}
	at := func(i int) point { return pts[(i%sides+sides)%sides] }
	pair := func(p point) string { return shapeNumber(p[0]) + " " + shapeNumber(p[1]) }
	cut := func(i, j int) point {
		p, q := at(i), at(j)
		return point{p[0] + float64((q[0]-p[0])*k), p[1] + float64((q[1]-p[1])*k)}
	}
	var b strings.Builder
	b.WriteString("M")
	b.WriteString(pair(cut(0, -1)))
	for i := 0; i < sides; i++ {
		b.WriteString("Q")
		b.WriteString(pair(at(i)))
		b.WriteByte(' ')
		b.WriteString(pair(cut(i, i+1)))
		if k < 0.5 {
			b.WriteString("L")
			b.WriteString(pair(cut(i+1, i)))
		}
	}
	b.WriteString("Z")
	return b.String()
}

func boxPath(cx, cy, rx, ry float64) string {
	l, r := shapeNumber(cx-rx), shapeNumber(cx+rx)
	return "M" + l + " " + shapeNumber(cy-ry) + "H" + r + "V" + shapeNumber(cy+ry) + "H" + l + "Z"
}

func taperPath(cx, cy, rx, ry, tip float64) string {
	t := math.Max(1.05, tip)
	tx := float64(rx * math.Sqrt(1-1/(t*t)))
	ty := cy - ry/t
	apex := cy - float64(t*ry)
	px := float64(tx * 0.14)
	py := ty + float64(0.86*(apex-ty))
	pair := func(x, y float64) string { return shapeNumber(x) + " " + shapeNumber(y) }
	return "M" + pair(cx-tx, ty) + "L" + pair(cx-px, py) + "Q" + pair(cx, apex) + " " + pair(cx+px, py) + "L" + pair(cx+tx, ty) + "Z"
}
