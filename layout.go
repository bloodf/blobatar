// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/styles/compose.ts and shapes.ts),
// MIT License. See NOTICE for the full license text.

package blobatar

import (
	"math"
	"strconv"
)

// Layout stays internal; Geometry exposes independent public endpoint values.
// These values are unrounded; only path serialization rounds coordinates.
// Keep explicit float64 conversions at multiply/add boundaries: they preserve
// the source rounding on targets that can fuse operations.
type layoutRegion struct {
	CX float64 `json:"cx"`
	CY float64 `json:"cy"`
	RX float64 `json:"rx"`
	RY float64 `json:"ry"`
}

type layoutEye struct {
	layoutRegion
	N   float64 `json:"n"`
	Rot float64 `json:"rot"`
}

type layoutBody struct {
	layoutEye
	Radii []float64 `json:"radii"`
	Sides int       `json:"sides,omitempty"`
	Round float64   `json:"round,omitempty"`
}

type layoutCircle struct {
	CX float64 `json:"cx"`
	CY float64 `json:"cy"`
	R  float64 `json:"r"`
}

type figureLayout struct {
	Shape  string         `json:"shape"`
	Body   layoutBody     `json:"body"`
	Face   layoutRegion   `json:"face"`
	Petals []layoutCircle `json:"petals"`
	Extra  []string       `json:"extra"`
	Eyes   [2]layoutEye   `json:"eyes"`
}

// composeLayout implements the frozen gen2 style: choose a silhouette, patch
// its body, establish its safe face region, decorate it, then fit the eyes.
func composeLayout(t *TraitReader) figureLayout {
	shape := shapeForValue(t.Float("shape"))
	core := 1.0
	switch shape {
	case "organic":
		core = 0.98
	case "boxy":
		core = 0.86
	case "capsule":
		core = 1.02
	case "nub":
		core = 0.88
	case "cloud", "droplet":
		core = 0.78
	case "hexagon":
		core = 1.05
	case "sun":
		core = 0.7
	case "triangle":
		core = 1.15
	}
	r := t.Num("body.r", 31, 38) * core
	body := layoutBody{
		layoutEye: layoutEye{
			layoutRegion: layoutRegion{
				CX: 50 + t.Jitter("body.x", 1.5), CY: 50 + t.Jitter("body.y", 1.5),
				RX: r, RY: r * t.Num("body.ratio", 0.92, 1.08),
			},
			N: t.Num("body.n", 1.9, 2.5),
		},
		Radii: make([]float64, t.Int("body.pts", 6, 8)),
	}
	for i := range body.Radii {
		body.Radii[i] = 1 + t.Jitter("body.r"+strconv.Itoa(i), 0.16)
	}
	switch shape {
	case "boxy":
		body.N = t.Num("body.n", 3.4, 6)
		body.Rot = t.Num("body.rot", -20, 20)
	case "capsule":
		body.RY *= t.Num("capsule.squat", 0.55, 0.68)
	case "droplet":
		body.CY += float64(0.22 * body.RY)
		body.N = 2
	case "hexagon", "triangle":
		body.Sides = 6
		body.Rot = t.Num("body.rot", -12, 12)
		if shape == "triangle" {
			body.Sides = 3
			body.Rot = t.Num("body.rot", -5, 5)
		}
		body.Round = t.Num("poly.round", 0.24, 0.5)
	}

	face := body.layoutRegion
	shrink := func(k float64) { face.RX = body.RX * k; face.RY = body.RY * k }
	switch shape {
	case "organic", "cloud":
		min := body.Radii[0]
		for _, v := range body.Radii[1:] {
			min = math.Min(min, v)
		}
		shrink(min * 0.95)
	case "capsule":
		shrink(0.94)
	case "droplet":
		face.CY += float64(body.RY * 0.05)
		shrink(0.88)
	case "hexagon":
		shrink(0.84)
	case "triangle":
		face.CY += float64(body.RY * 0.1)
		face.RX = body.RX * 0.54
		face.RY = body.RY * 0.36
	}

	l := figureLayout{Shape: shape, Body: body, Face: face, Petals: []layoutCircle{}, Extra: []string{}}
	switch shape {
	case "capsule":
		for _, s := range []float64{-1, 1} {
			l.Petals = append(l.Petals, layoutCircle{body.CX + float64(s*(body.RX-body.RY)), body.CY, body.RY})
		}
	case "nub":
		count := t.Int("nub.n", 1, 2)
		for i := 0; i < count; i++ {
			a := t.Num("nub.a"+strconv.Itoa(i), 0, 2*math.Pi)
			l.Petals = append(l.Petals, layoutCircle{
				body.CX + float64(math.Cos(a)*body.RX*0.88),
				body.CY + float64(math.Sin(a)*body.RX*0.88),
				body.RX * t.Num("nub.r"+strconv.Itoa(i), 0.24, 0.4),
			})
		}
	case "cloud":
		count := t.Int("cloud.n", 4, 6)
		for i := 0; i < count; i++ {
			a := math.Pi + math.Pi*(float64(i)+0.5)/float64(count)
			l.Petals = append(l.Petals, layoutCircle{
				body.CX + float64(math.Cos(a)*body.RX*0.8),
				body.CY + float64(math.Sin(a)*body.RX*0.5),
				body.RX * t.Num("cloud.r"+strconv.Itoa(i), 0.44, 0.62),
			})
		}
	case "droplet":
		l.Extra = append(l.Extra, taperPath(body.CX, body.CY, body.RX, body.RY, t.Num("droplet.tip", 1.4, 1.65)))
	case "sun":
		count := t.Int("sun.n", 6, 9)
		dist := body.RX * t.Num("sun.dist", 1, 1.08)
		pr := body.RX * t.Num("sun.r", 0.2, 0.26)
		off := t.Num("sun.rot", 0, 2*math.Pi)
		for i := 0; i < count; i++ {
			a := off + 2*math.Pi*float64(i)/float64(count)
			l.Petals = append(l.Petals, layoutCircle{body.CX + float64(math.Cos(a)*dist), body.CY + float64(math.Sin(a)*dist), pr})
		}
	}
	l.Eyes = faceFit(t, body, face)
	return l
}

// faceFit keeps the asymmetric eye cluster inside the silhouette's face region.
func faceFit(t *TraitReader, b layoutBody, face layoutRegion) [2]layoutEye {
	rx := b.RX
	er0 := t.Num("eye.rx", 0.075, 0.105) * rx
	ratio := t.Num("eye.ratio", 1.9, 3.2)
	scale := t.Num("eye.scale", 0.78, 1.24)
	stretch := t.Num("eye.stretch", 0.85, 1.18)
	clearance := t.Num("eye.gap", 0.1, 0.24) * rx
	wide := er0 * math.Max(1, scale)
	tall := er0 * ratio * math.Max(1, scale*stretch)
	gap0 := float64(wide+float64(rx*0.03)) + clearance

	gx := t.Jitter("gaze.x", 0.09) * face.RX
	gy := t.Num("gaze.y", -0.2, 0.08) * face.RY
	dy := t.Jitter("eye.dy", 0.04) * face.RY
	reach := math.Hypot(wide, tall)
	need := math.Hypot((math.Abs(gx)+gap0+reach)/face.RX, (math.Abs(gy)+math.Abs(dy)+reach)/face.RY)
	fit := 1.0
	if need > 0.9 {
		fit = 0.9 / need
	}

	er := er0 * fit
	eyeRY := er * ratio
	gap := gap0 * fit
	room := math.Max(0, math.Min(1, clearance/tall))
	bound := math.Min(12, math.Asin(room)*180/math.Pi)
	lean := t.Num("eye.lean", -1, 1) * bound
	lean2 := math.Max(-12, math.Min(12, lean+t.Jitter("eye.lean2", 3.5)))
	cx, cy := face.CX+float64(gx*fit), face.CY+float64(gy*fit)
	return [2]layoutEye{
		{layoutRegion{cx - gap, cy, er, eyeRY}, t.Num("eye.n", 3.5, 6), lean},
		{layoutRegion{cx + gap, cy + float64(dy*fit), er * scale, eyeRY * scale * stretch}, t.Num("eye.n", 3.5, 6), lean2},
	}
}

func (e layoutEye) path() string {
	return superellipse(e.CX, e.CY, e.RX, e.RY, e.N, e.Rot)
}

func (l figureLayout) corePath() string {
	b := l.Body
	switch l.Shape {
	case "organic", "cloud":
		return blobPath(b.CX, b.CY, b.RX, b.RY, b.Radii, b.Rot)
	case "capsule":
		return boxPath(b.CX, b.CY, b.RX-b.RY, b.RY)
	case "hexagon", "triangle":
		return polygonPath(b.CX, b.CY, b.RX, b.RY, b.Sides, b.Round, b.Rot)
	default:
		return b.path()
	}
}
