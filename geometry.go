// Copyright (c) 2026 Alain
//
// Ported from Beans packages/beans-blobatar/geometry.ts and frame.ts, and
// Alain00/blobatar vendor/core/animate.ts and render.ts. MIT License.
// See NOTICE for the full license text.

package blobatar

import (
	"fmt"
	"math"
	"strings"
)

// Fixed contour sizes and decoration slots used by the pinned geometry format.
const (
	CoreSegments       = 24
	ExtraSegments      = 4
	BackgroundSegments = 8
	PetalSlots         = 9
)

// AvatarPoint is an x/y coordinate in the 100-by-100 avatar view box.
type AvatarPoint [2]float64

// AvatarCubic is one cubic after the preceding endpoint: c1x c1y c2x c2y x y.
type AvatarCubic [6]float64

// AvatarCubicPath is a closed contour, clockwise in screen coordinates.
type AvatarCubicPath struct {
	Start    AvatarPoint   `json:"start"`
	Segments []AvatarCubic `json:"segments"`
}

// AvatarCircle is a silhouette decoration. An unused slot has radius zero.
type AvatarCircle struct {
	CX float64 `json:"cx"`
	CY float64 `json:"cy"`
	R  float64 `json:"r"`
}

// AvatarEye describes an unposed superellipse; Rot is measured in degrees.
type AvatarEye struct {
	CX  float64 `json:"cx"`
	CY  float64 `json:"cy"`
	RX  float64 `json:"rx"`
	RY  float64 `json:"ry"`
	N   float64 `json:"n"`
	Rot float64 `json:"rot"`
}

// AvatarBackground retains a circle contour at opacity zero when absent.
type AvatarBackground struct {
	Path    AvatarCubicPath `json:"path"`
	Fill    string          `json:"fill"`
	Opacity float64         `json:"opacity"`
}

// AvatarPalette contains resolved uppercase #RRGGBB colors. Head and Eye
// include the expression's tint; BG never changes with the expression.
type AvatarPalette struct {
	Head string `json:"head"`
	Eye  string `json:"eye"`
	BG   string `json:"bg"`
}

// IdleSeeds holds deterministic motion periods, offsets (milliseconds), and
// glance distances. Geometry carries these even when Motion is false.
type IdleSeeds struct {
	Phase        float64 `json:"phase"`
	Bob          float64 `json:"bob"`
	Blink        float64 `json:"blink"`
	BlinkPhase   float64 `json:"blinkPhase"`
	Saccade      float64 `json:"saccade"`
	SaccadePhase float64 `json:"saccadePhase"`
	LookX        float64 `json:"lookX"`
	LookY        float64 `json:"lookY"`
	LookMX       float64 `json:"lookMX"`
	LookMY       float64 `json:"lookMY"`
}

// AvatarGeometry is a version-one endpoint matching Beans botAvatarGeometry.
// Eyes remain unposed: consumers apply Pose separately, exactly once. Each
// call owns its contour slices, so callers can safely keep and edit endpoints.
type AvatarGeometry struct {
	Version    int                      `json:"version"`
	Background AvatarBackground         `json:"background"`
	Core       AvatarCubicPath          `json:"core"`
	Extra      AvatarCubicPath          `json:"extra"`
	Petals     [PetalSlots]AvatarCircle `json:"petals"`
	Eyes       [2]AvatarEye             `json:"eyes"`
	Pose       Pose                     `json:"pose"`
	Palette    AvatarPalette            `json:"palette"`
	Motion     bool                     `json:"motion"`
	Seeds      IdleSeeds                `json:"seeds"`
}

// shapePosition matches appearance.ts's explicit shape overrides. An omitted
// shape keeps the seed's original trait instead of introducing an override.
var shapePosition = map[string]float64{
	"round": 0.11, "organic": 0.35, "boxy": 0.54, "capsule": 0.65, "nub": 0.745,
	"cloud": 0.825, "droplet": 0.8875, "hexagon": 0.9325, "sun": 0.965, "triangle": 0.99,
}

// Geometry resolves a seed, optional Look and state into numeric endpoint
// geometry. Its accepted look inputs and validation match ResolveAppearance:
// nil (including *Look(nil)) means seeded defaults, and state "" means no
// state override. Unknown states and invalid looks return an error.
//
// This does not render SVG or evaluate animation frames.
func Geometry(botID string, look any, state string) (AvatarGeometry, error) {
	app, err := resolveLook(look, state)
	if err != nil {
		return AvatarGeometry{}, err
	}
	resolved, err := resolveFromAppearance(botID, app)
	if err != nil {
		return AvatarGeometry{}, err
	}
	overrides := TraitOverrides{}
	if app.Shape != "" {
		overrides["shape"] = FixedTrait(shapePosition[app.Shape])
	}
	t := NewTraits(botID, true, overrides)
	l := composeLayout(t)
	expr, err := expressionForName(resolved.Expression)
	if err != nil {
		return AvatarGeometry{}, err
	}
	pal, err := expressionPalette(Palette{Head: resolved.Palette.Head, Eye: resolved.Palette.Eye, BG: resolved.Palette.Bg}, expr)
	if err != nil {
		return AvatarGeometry{}, err
	}
	centre := point{l.Body.CX, l.Body.CY}
	core, err := geometryContour(l.corePath(), CoreSegments, centre)
	if err != nil {
		return AvatarGeometry{}, fmt.Errorf("blobatar: core: %w", err)
	}
	if len(l.Extra) > 1 {
		return AvatarGeometry{}, fmt.Errorf("blobatar: more than one extra outline")
	}
	extra := collapsedContour(ExtraSegments, centre)
	if len(l.Extra) == 1 {
		extra, err = geometryContour(l.Extra[0], ExtraSegments, centre)
		if err != nil {
			return AvatarGeometry{}, fmt.Errorf("blobatar: extra: %w", err)
		}
	}
	backgroundD := superellipse(50, 50, 50, 50, 2, 0)
	opacity := 1.0
	switch resolved.Background {
	case "none":
		opacity = 0
	case "square":
		backgroundD = "M0 0H100V100H0Z"
	case "squircle":
		backgroundD = superellipse(50, 50, 50, 50, 6, 0)
	}
	background, err := geometryContour(backgroundD, BackgroundSegments, point{50, 50})
	if err != nil {
		return AvatarGeometry{}, fmt.Errorf("blobatar: background: %w", err)
	}
	g := AvatarGeometry{
		Version: 1, Background: AvatarBackground{Path: background, Fill: resolved.Palette.Bg, Opacity: opacity},
		Core: core, Extra: extra, Pose: expr.P, Motion: resolved.Motion, Seeds: motionSeeds(t),
		Palette: AvatarPalette{Head: strings.ToUpper(pal.Head), Eye: strings.ToUpper(pal.Eye), BG: resolved.Palette.Bg},
	}
	if len(l.Petals) > PetalSlots {
		return AvatarGeometry{}, fmt.Errorf("blobatar: %d petals exceed %d slots", len(l.Petals), PetalSlots)
	}
	for i := range g.Petals {
		g.Petals[i] = AvatarCircle{CX: centre[0], CY: centre[1]}
	}
	for i, p := range l.Petals {
		g.Petals[i] = AvatarCircle{CX: roundGeometry(p.CX*100) / 100, CY: roundGeometry(p.CY*100) / 100, R: roundGeometry(p.R*100) / 100}
	}
	for i, e := range l.Eyes {
		g.Eyes[i] = AvatarEye{CX: e.CX, CY: e.CY, RX: e.RX, RY: e.RY, N: e.N, Rot: e.Rot}
	}
	return g, nil
}

func geometryContour(path string, count int, centre point) (AvatarCubicPath, error) {
	curves, err := pathCubics(path)
	if err != nil {
		return AvatarCubicPath{}, err
	}
	p, err := normalizeContour(curves, count, centre)
	if err != nil {
		return AvatarCubicPath{}, err
	}
	out := AvatarCubicPath{Start: AvatarPoint(p.Start), Segments: make([]AvatarCubic, len(p.Segments))}
	for i, segment := range p.Segments {
		out.Segments[i] = AvatarCubic(segment)
	}
	return out, nil
}

func collapsedContour(count int, centre point) AvatarCubicPath {
	out := AvatarCubicPath{Start: AvatarPoint(centre), Segments: make([]AvatarCubic, count)}
	for i := range out.Segments {
		out.Segments[i] = AvatarCubic{centre[0], centre[1], centre[0], centre[1], centre[0], centre[1]}
	}
	return out
}

// Math.round ties toward positive infinity, unlike Go's math.Round. Values
// here are finite and bounded by the authored layout/motion trait ranges.
func roundGeometry(v float64) float64 {
	// The explicit conversion also protects callers when this helper is
	// inlined after a scaling multiplication (petals and motion seeds).
	whole, fraction := math.Modf(float64(v))
	if fraction >= 0.5 {
		return whole + 1
	}
	if fraction < -0.5 {
		return whole - 1
	}
	return whole
}

func motionSeeds(t *TraitReader) IdleSeeds {
	blink := roundGeometry(t.Num("motion.blink", 3500, 6500))
	saccade := roundGeometry(t.Num("motion.saccade", 4200, 7600))
	x := roundGeometry(t.Num("motion.lookX", 1, 2.2)*100) / 100
	y := roundGeometry(t.Num("motion.lookY", 0.8, 1.7)*100) / 100
	xSign, ySign := 1.0, 1.0
	if t.Bool("motion.lookXFlip") {
		xSign = -1
	}
	if t.Bool("motion.lookYFlip") {
		ySign = -1
	}
	return IdleSeeds{
		Phase: roundGeometry(t.Num("motion.phase", 0, 2800)), Bob: roundGeometry(t.Num("motion.bob", 0, 3400)),
		Blink: blink, BlinkPhase: roundGeometry(t.Num("motion.blinkPhase", 0, blink)),
		Saccade: saccade, SaccadePhase: roundGeometry(t.Num("motion.saccadePhase", 0, saccade)),
		LookX: x * xSign, LookY: y * ySign, LookMX: x, LookMY: y,
	}
}
