// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/expression.ts and morph.ts),
// MIT License. See NOTICE for the full license text.

package blobatar

import (
	"fmt"
	"math"
	"strconv"
)

// Pose contains the expression channels carried by AvatarGeometry. Scales are
// factors, tilts are degrees, offsets are view-box units, and Heat/Shake/Rock
// are amplitudes. The *2 fields are right-eye differentials, not endpoints.
// The zero value is not an idle pose: idle has ESX and ESY equal to one.
type Pose struct {
	ESX   float64 `json:"esx"`
	ESY   float64 `json:"esy"`
	Tilt  float64 `json:"tilt"`
	EDY   float64 `json:"edy"`
	EDX   float64 `json:"edx"`
	ESX2  float64 `json:"esx2"`
	ESY2  float64 `json:"esy2"`
	Tilt2 float64 `json:"tilt2"`
	EDY2  float64 `json:"edy2"`
	Lock  float64 `json:"lock"`
	Heat  float64 `json:"heat"`
	Shake float64 `json:"shake"`
	Rock  float64 `json:"rock"`
	BDY   float64 `json:"bdy"`
}

type pose = Pose

func identityPose() pose { return pose{ESX: 1, ESY: 1} }

type expression struct {
	P    pose
	tint string
}

// The roster is frozen to the Beans source pinned in testdata/golden/SOURCE.
// Lookup returns values, so applying a pose never mutates the authored table.
var expressionRoster = map[string]expression{
	"idle": {P: identityPose()},
	"happy": {P: pose{ESX: 1.72, ESY: 0.3, Tilt: 8, EDY: -1.5, EDX: 1.5,
		ESX2: 0.08, ESY2: 0.05, Tilt2: -16, Lock: 1, BDY: -2.2}},
	"sad": {P: pose{ESX: 0.6, ESY: 0.56, Tilt: 26, EDY: 3.6, EDX: 1.9,
		ESX2: -0.05, ESY2: -0.07, Tilt2: -7, Lock: 1, BDY: 2.6}},
	"mad": {P: pose{ESX: 1.85, ESY: 0.26, Tilt: -33, EDY: 0.4, EDX: 0.6,
		ESY2: -0.03, Tilt2: 5, Lock: 1, Heat: 0.62, Shake: 0.55, BDY: 0.8}, tint: "hot"},
	"surprised": {P: pose{ESX: 1.34, ESY: 1.2, Tilt: -6, EDY: -1.05, EDX: 0.5,
		ESX2: 0.05, ESY2: 0.07, Tilt2: 3, Lock: 1, BDY: -1.4}},
	"wink": {P: pose{ESX: 1.32, ESY: 0.76, Tilt: 5, EDY: -0.6, EDX: 0.8,
		ESX2: 0.26, ESY2: -0.56, Tilt2: -11, Lock: 1, BDY: -1.1}},
	"sleepy": {P: pose{ESX: 1.14, ESY: 0.22, EDY: 2.4, EDX: 0.3,
		ESX2: -0.04, ESY2: 0.03, Tilt2: 4, Lock: 1, BDY: 1.2}},
	"smug": {P: pose{ESX: 1.3, ESY: 0.42, Tilt: 18, EDY: -0.5, EDX: 0.5,
		ESX2: 0.06, ESY2: -0.06, Tilt2: -36, Lock: 1, BDY: -1}},
	"unsure": {P: pose{ESX: 0.95, ESY: 1.02, Tilt: 4, EDY: -0.2, EDX: 0.3,
		ESX2: 0.24, ESY2: -0.44, Tilt2: -18, Lock: 1}},
	"scared": {P: pose{ESX: 0.78, ESY: 0.96, Tilt: -12, EDY: -1.5, EDX: -0.8,
		ESX2: -0.04, ESY2: 0.05, Tilt2: 4, Lock: 1, Shake: 0.35, BDY: -0.6}},
	"love": {P: pose{ESX: 0.86, ESY: 1.28, Tilt: -14, EDY: -0.5, EDX: -0.35,
		ESX2: 0.05, ESY2: 0.06, Tilt2: 6, Lock: 1, Heat: 0.6, BDY: -1.6}, tint: "rose"},
	"shy": {P: pose{ESX: 0.62, ESY: 0.5, Tilt: 10, EDY: 1.4, EDX: -0.2,
		ESX2: -0.05, ESY2: -0.04, Tilt2: -8, Lock: 1, Heat: 0.55, BDY: 0.9}, tint: "blush"},
	"sick": {P: pose{ESX: 1.25, ESY: 0.34, Tilt: 20, EDY: 1.8, EDX: 0.8,
		ESX2: 0.05, ESY2: -0.05, Tilt2: -6, Lock: 1, Heat: 0.6, Shake: 0.18, BDY: 1.4}, tint: "bile"},
	"thinking": {P: pose{ESX: 1.15, ESY: 0.62, EDY: 4.2, EDX: 0.4,
		ESX2: 0.02, ESY2: 0.06, EDY2: -8.4, Lock: 1, Rock: 0.8, BDY: -0.4}},
}

func expressionForName(name string) (expression, error) {
	if name == "" {
		name = "idle"
	}
	e, ok := expressionRoster[name]
	if !ok {
		return expression{}, fmt.Errorf("blobatar: unknown expression %q", name)
	}
	return e, nil
}

// poseNumber matches JS r3, including half ties toward positive infinity and
// negative-zero removal. It is used only with finite authored/derived poses.
func poseNumber(v float64) string {
	// Keep the multiplication rounded to binary64 before an inlined Modf
	// subtracts the integer part; otherwise arm64 may fuse them into FMA.
	whole, fraction := math.Modf(float64(v * 1000))
	if fraction >= 0.5 {
		whole++
	} else if fraction < -0.5 {
		whole--
	}
	if whole == 0 {
		return "0"
	}
	return strconv.FormatFloat(whole/1000, 'f', -1, 64)
}

// poseVars omits identity channels and heat, which is resolved in the palette.
func poseVars(p pose) map[string]string {
	out := map[string]string{}
	for _, c := range []struct {
		name           string
		value, initial float64
	}{
		{"esx", p.ESX, 1}, {"esy", p.ESY, 1}, {"tilt", p.Tilt, 0},
		{"edy", p.EDY, 0}, {"edx", p.EDX, 0}, {"esx2", p.ESX2, 0},
		{"esy2", p.ESY2, 0}, {"tilt2", p.Tilt2, 0}, {"edy2", p.EDY2, 0},
		{"lock", p.Lock, 0}, {"shake", p.Shake, 0}, {"rock", p.Rock, 0}, {"bdy", p.BDY, 0},
	} {
		if c.value != c.initial {
			out["--mo-"+c.name] = poseNumber(c.value)
		}
	}
	return out
}

// bakePose changes only the returned eye values. The body translation belongs
// outside the figure and never inside the background. Shake and rock remain
// loop amplitudes, not static transforms.
func bakePose(l figureLayout, p pose) (figureLayout, string) {
	for i, e := range l.Eyes {
		wrap, sel := -1.0, 0.0
		if i != 0 {
			wrap, sel = 1, 1
		}
		e.CX += float64(p.EDX * wrap)
		e.CY = e.CY + p.EDY + float64(sel*p.EDY2)
		e.RX *= p.ESX + float64(sel*p.ESX2)
		e.RY *= p.ESY + float64(sel*p.ESY2)
		e.Rot = float64(e.Rot*(1-p.Lock)) + float64((p.Tilt+float64(sel*p.Tilt2))*wrap)
		l.Eyes[i] = e
	}
	if p.BDY != 0 {
		return l, "translate(0 " + poseNumber(p.BDY) + ")"
	}
	return l, ""
}

type poseTransform struct {
	Eyes [2]string `json:"eyes"`
	Wrap string    `json:"wrap"`
}

// poseTransforms represents the same geometry as bakePose without redrawing
// eye paths. rockPhase=1 is the static endpoint; moving phases drive the seesaw.
func poseTransforms(l figureLayout, p pose, rockPhase float64) poseTransform {
	out := poseTransform{Wrap: "translate(0 " + poseNumber(p.BDY) + ")"}
	for i, e := range l.Eyes {
		wrap, sel := -1.0, 0.0
		if i != 0 {
			wrap, sel = 1, 1
		}
		ph := float64(sel*(1-p.Rock)) + float64(p.Rock*((1+float64(wrap*rockPhase))/2))
		out.Eyes[i] = "translate(" + poseNumber(e.CX+float64(p.EDX*wrap)) + " " + poseNumber(e.CY+p.EDY+float64(ph*p.EDY2)) + ")" +
			" rotate(" + poseNumber(float64((p.Tilt+float64(sel*p.Tilt2))*wrap)+float64(e.Rot*(1-p.Lock))) + ")" +
			" scale(" + poseNumber(p.ESX+float64(sel*p.ESX2)) + " " + poseNumber(p.ESY+float64(sel*p.ESY2)) + ")" +
			" rotate(" + poseNumber(-e.Rot) + ") translate(" + poseNumber(-e.CX) + " " + poseNumber(-e.CY) + ")"
	}
	return out
}

// lerpPose follows the source's unclamped interpolation, with nil as identity.
// Weighted endpoints preserve exact equality at 0 and 1. Explicit conversions
// prevent an FMA from changing the source's two-product rounding.
func lerpPose(a, b *pose, t float64) pose {
	from, to := identityPose(), identityPose()
	if a != nil {
		from = *a
	}
	if b != nil {
		to = *b
	}
	m := func(a, b float64) float64 { return float64(a*(1-t)) + float64(b*t) }
	return pose{
		ESX: m(from.ESX, to.ESX), ESY: m(from.ESY, to.ESY), Tilt: m(from.Tilt, to.Tilt),
		EDY: m(from.EDY, to.EDY), EDX: m(from.EDX, to.EDX), ESX2: m(from.ESX2, to.ESX2),
		ESY2: m(from.ESY2, to.ESY2), Tilt2: m(from.Tilt2, to.Tilt2), EDY2: m(from.EDY2, to.EDY2),
		Lock: m(from.Lock, to.Lock), Heat: m(from.Heat, to.Heat), Shake: m(from.Shake, to.Shake),
		Rock: m(from.Rock, to.Rock), BDY: m(from.BDY, to.BDY),
	}
}

// expressionPalette applies only the four tinting expressions. Cool poses
// leave the input palette byte-for-byte intact, including its background.
func expressionPalette(pal Palette, e expression) (Palette, error) {
	var target Tint
	switch e.tint {
	case "":
		return pal, nil
	case "hot":
		target = HotTint()
	case "rose":
		target = RoseTint()
	case "blush":
		target = BlushTint()
	case "bile":
		target = BileTint()
	default:
		return Palette{}, fmt.Errorf("blobatar: unknown expression tint %q", e.tint)
	}
	head, eye, err := Tinted(pal.Head, pal.Eye, target)
	if err != nil {
		return Palette{}, err
	}
	head, err = MixHex(pal.Head, head, e.P.Heat)
	if err != nil {
		return Palette{}, err
	}
	eye, err = MixHex(pal.Eye, eye, e.P.Heat)
	if err != nil {
		return Palette{}, err
	}
	pal.Head, pal.Eye = head, eye
	return pal, nil
}
