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
	n := math.Floor(v*100+0.5) / 100
	if n == 0 {
		return "0"
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}

// superellipse emits the four cubic quadrants used by the pinned source.
func superellipse(cx, cy, rx, ry, n, rot float64) string {
	k := math.Min(1, (8*math.Pow(2, -1/n)-4)/3)
	ak, bk := rx*k, ry*k
	pts := []point{{rx, 0}, {rx, bk}, {ak, ry}, {0, ry}, {-ak, ry}, {-rx, bk}, {-rx, 0}, {-rx, -bk}, {-ak, -ry}, {0, -ry}, {ak, -ry}, {rx, -bk}, {rx, 0}}
	angle := rot * math.Pi / 180
	cos, sin := math.Cos(angle), math.Sin(angle)
	at := func(i int) string {
		p := pts[i]
		return shapeNumber(cx+p[0]*cos-p[1]*sin) + " " + shapeNumber(cy+p[0]*sin+p[1]*cos)
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
