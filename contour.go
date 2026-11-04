// Copyright (c) 2026 Alain
//
// Ported from Beans packages/beans-blobatar/geometry.ts, which derives from
// Alain00/blobatar. MIT License; see NOTICE for the full license text.

package blobatar

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type point [2]float64
type bezier [4]point
type cubicPath struct {
	Start    point
	Segments [][6]float64
}

var pathTokens = regexp.MustCompile(`[MCQLHVZmcqlhvz]|-?\d*\.?\d+(?:[eE][-+]?\d+)?`)

// pathCubics converts the absolute SVG commands emitted by the pinned source
// into cubic curves. It rejects unexpected input rather than silently dropping it.
func pathCubics(d string) ([]bezier, error) {
	indexes := pathTokens.FindAllStringIndex(d, -1)
	tokens := make([]string, 0, len(indexes))
	last := 0
	for _, idx := range indexes {
		if strings.Trim(strings.TrimSpace(d[last:idx[0]]), ",") != "" {
			return nil, fmt.Errorf("invalid path near %q", d[last:idx[0]])
		}
		tokens = append(tokens, d[idx[0]:idx[1]])
		last = idx[1]
	}
	if strings.Trim(strings.TrimSpace(d[last:]), ",") != "" {
		return nil, fmt.Errorf("invalid path near %q", d[last:])
	}
	var out []bezier
	var cur, start point
	cmd := ""
	i := 0
	number := func() (float64, error) {
		if i >= len(tokens) {
			return 0, fmt.Errorf("missing coordinate")
		}
		n, err := strconv.ParseFloat(tokens[i], 64)
		i++
		if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
			return 0, fmt.Errorf("invalid coordinate")
		}
		return n, nil
	}
	readPoint := func() (point, error) {
		x, e := number()
		if e != nil {
			return point{}, e
		}
		y, e := number()
		return point{x, y}, e
	}
	line := func(to point) {
		if to == cur {
			return
		}
		dx, dy := to[0]-cur[0], to[1]-cur[1]
		out = append(out, bezier{cur, {cur[0] + dx/3, cur[1] + dy/3}, {cur[0] + 2*dx/3, cur[1] + 2*dy/3}, to})
		cur = to
	}
	for i < len(tokens) {
		if len(tokens[i]) == 1 && strings.ContainsAny(tokens[i], "MCQLHVZmcqlhvz") {
			cmd = tokens[i]
			i++
		}
		switch cmd {
		case "M":
			p, e := readPoint()
			if e != nil {
				return nil, e
			}
			cur, start = p, p
			cmd = "L"
		case "C":
			a, e := readPoint()
			if e != nil {
				return nil, e
			}
			b, e := readPoint()
			if e != nil {
				return nil, e
			}
			p, e := readPoint()
			if e != nil {
				return nil, e
			}
			out = append(out, bezier{cur, a, b, p})
			cur = p
		case "Q":
			q, e := readPoint()
			if e != nil {
				return nil, e
			}
			p, e := readPoint()
			if e != nil {
				return nil, e
			}
			out = append(out, bezier{cur, {cur[0] + 2.0/3*(q[0]-cur[0]), cur[1] + 2.0/3*(q[1]-cur[1])}, {p[0] + 2.0/3*(q[0]-p[0]), p[1] + 2.0/3*(q[1]-p[1])}, p})
			cur = p
		case "L":
			p, e := readPoint()
			if e != nil {
				return nil, e
			}
			line(p)
		case "H":
			x, e := number()
			if e != nil {
				return nil, e
			}
			line(point{x, cur[1]})
		case "V":
			y, e := number()
			if e != nil {
				return nil, e
			}
			line(point{cur[0], y})
		case "Z":
			line(start)
			cmd = ""
		default:
			return nil, fmt.Errorf("unsupported path command %q", cmd)
		}
	}
	return out, nil
}

func bezierAt(b bezier, t float64) point {
	s := 1 - t
	return point{s*s*s*b[0][0] + 3*s*s*t*b[1][0] + 3*s*t*t*b[2][0] + t*t*t*b[3][0], s*s*s*b[0][1] + 3*s*s*t*b[1][1] + 3*s*t*t*b[2][1] + t*t*t*b[3][1]}
}
func splitBezier(c bezier, t float64) (bezier, bezier) {
	lerp := func(p, q point) point { return point{p[0] + (q[0]-p[0])*t, p[1] + (q[1]-p[1])*t} }
	ab, bc, cd := lerp(c[0], c[1]), lerp(c[1], c[2]), lerp(c[2], c[3])
	abc, bcd := lerp(ab, bc), lerp(bc, cd)
	m := lerp(abc, bcd)
	return bezier{c[0], ab, abc, m}, bezier{m, bcd, cd, c[3]}
}

// normalizeContour preserves every source boundary and subdivides by angular
// sweep, matching the TypeScript generator's ordering and tie tolerances.
func normalizeContour(src []bezier, count int, centre point) (cubicPath, error) {
	if len(src) == 0 || len(src) > count {
		return cubicPath{}, fmt.Errorf("contour has %d segments; need 1..%d", len(src), count)
	}
	area := 0.0
	for _, b := range src {
		area += b[0][0]*b[3][1] - b[3][0]*b[0][1]
	}
	pieces := append([]bezier(nil), src...)
	if area < 0 {
		for i, j := 0, len(pieces)-1; i <= j; i, j = i+1, j-1 {
			a, b := pieces[i], pieces[j]
			pieces[i] = bezier{b[3], b[2], b[1], b[0]}
			pieces[j] = bezier{a[3], a[2], a[1], a[0]}
		}
	}
	angle := func(p point) float64 { return math.Atan2(p[1]-centre[1], p[0]-centre[0]) }
	sweep := func(from, to float64) float64 { d := to - from; return d - 2*math.Pi*math.Floor(d/(2*math.Pi)) }
	span := func(b bezier) float64 {
		a := angle(b[0])
		s := sweep(a, angle(b[3]))
		if s > math.Pi && sweep(a, angle(bezierAt(b, 0.5))) > s {
			return 0
		}
		return s
	}
	for len(pieces) < count {
		widest, most := 0, -1.0
		for i, b := range pieces {
			s := span(b)
			if s > most+1e-12 {
				most = s
				widest = i
			}
		}
		b := pieces[widest]
		t := 0.5
		if most > 1e-9 {
			a, lo, hi := angle(b[0]), 0.0, 1.0
			for k := 0; k < 60; k++ {
				mid := (lo + hi) / 2
				if sweep(a, angle(bezierAt(b, mid))) < most/2 {
					lo = mid
				} else {
					hi = mid
				}
			}
			t = (lo + hi) / 2
		}
		left, right := splitBezier(b, t)
		pieces = append(pieces, bezier{})
		copy(pieces[widest+2:], pieces[widest+1:])
		pieces[widest], pieces[widest+1] = left, right
	}
	first, best := 0, math.Inf(1)
	for i, p := range pieces {
		s := sweep(-math.Pi/2, angle(p[0]))
		dist := math.Min(s, 2*math.Pi-s)
		if dist < best-1e-9 {
			first, best = i, dist
		}
	}
	ordered := append(append([]bezier(nil), pieces[first:]...), pieces[:first]...)
	result := cubicPath{Start: ordered[0][0], Segments: make([][6]float64, 0, len(ordered))}
	for _, p := range ordered {
		result.Segments = append(result.Segments, [6]float64{p[1][0], p[1][1], p[2][0], p[2][1], p[3][0], p[3][1]})
	}
	return result, nil
}
