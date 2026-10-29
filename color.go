// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/color.ts), MIT License.
// See NOTICE for the full license text.

package blobatar

import (
	"fmt"
	"math"
	"strconv"
)

// Oklch is a color in OKLCh space.
type Oklch struct {
	L float64
	C float64
	H float64
}

// ColorKey is a palette slot.
type ColorKey string

const (
	ColorBG   ColorKey = "bg"
	ColorHead ColorKey = "head"
	ColorEye  ColorKey = "eye"
)

// Palette is every color slot a blobatar has, as #rrggbb hex (lowercase).
type Palette struct {
	BG   string `json:"bg,omitempty"`
	Head string `json:"head,omitempty"`
	Eye  string `json:"eye,omitempty"`
}

// toLinear converts OKLCh to linear-light sRGB. Components may fall outside [0,1].
func toLinear(color Oklch) [3]float64 {
	r := (color.H * math.Pi) / 180
	a := color.C * math.Cos(r)
	b := color.C * math.Sin(r)

	l_ := color.L + 0.3963377774*a + 0.2158037573*b
	m_ := color.L - 0.1055613458*a - 0.0638541728*b
	s_ := color.L - 0.0894841775*a - 1.291485548*b

	L := l_ * l_ * l_
	M := m_ * m_ * m_
	S := s_ * s_ * s_

	return [3]float64{
		4.0767416621*L - 3.3077115913*M + 0.2309699292*S,
		-1.2684380046*L + 2.6097574011*M - 0.3413193965*S,
		-0.0041960863*L - 0.7034186147*M + 1.707614701*S,
	}
}

// inGamut reports whether linear-light sRGB components are in gamut.
// NaN and ±Inf are out of gamut (explicit check, plus TS-style
// !(v >= lo && v <= hi) which also rejects non-finite values).
func inGamut(rgb [3]float64) bool {
	for _, v := range rgb {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
		if !(v >= -1e-4 && v <= 1+1e-4) {
			return false
		}
	}
	return true
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// resolveColor resolves to in-gamut linear sRGB, reducing chroma if needed.
func resolveColor(color Oklch) [3]float64 {
	rgb := toLinear(color)
	if !inGamut(rgb) {
		lo := 0.0
		hi := color.C
		for i := 0; i < 12; i++ {
			mid := (lo + hi) / 2
			if inGamut(toLinear(Oklch{L: color.L, C: mid, H: color.H})) {
				lo = mid
			} else {
				hi = mid
			}
		}
		rgb = toLinear(Oklch{L: color.L, C: lo, H: color.H})
	}
	for i := range rgb {
		if !isFinite(rgb[i]) {
			rgb[i] = 0
			continue
		}
		if rgb[i] < 0 {
			rgb[i] = 0
		} else if rgb[i] > 1 {
			rgb[i] = 1
		}
	}
	return rgb
}

func luminance(color Oklch) float64 {
	rgb := resolveColor(color)
	return 0.2126*rgb[0] + 0.7152*rgb[1] + 0.0722*rgb[2]
}

// Contrast is the WCAG contrast ratio between a and b.
func Contrast(a, b Oklch) float64 {
	x := luminance(a)
	y := luminance(b)
	return (math.Max(x, y) + 0.05) / (math.Min(x, y) + 0.05)
}

// EnsureContrast pushes fg's lightness away from bg until the pair clears min.
func EnsureContrast(fg, bg Oklch, min float64) Oklch {
	if Contrast(fg, bg) >= min {
		return fg
	}

	lean := 1.0
	if fg.L < bg.L {
		lean = -1
	}
	for _, dir := range []float64{lean, -lean} {
		probe := fg
		for i := 0; i < 60; i++ {
			probe.L = math.Min(1, math.Max(0, probe.L+dir*0.02))
			if Contrast(probe, bg) >= min {
				return probe
			}
			if probe.L == 0 || probe.L == 1 {
				break
			}
		}
	}

	black := Oklch{L: 0, C: 0, H: fg.H}
	white := Oklch{L: 1, C: 0, H: fg.H}
	if Contrast(black, bg) >= Contrast(white, bg) {
		return black
	}
	return white
}

// ToHex encodes color as lowercase #rrggbb.
//
// Non-finite OKLCh components (NaN, ±Inf) return an error — never a silent
// "#000000" and never int(math.Round(NaN)), which is platform-dependent.
// Callers that need a hex string must handle the error (MixHex/Tinted/
// PaletteColors propagate or surface it).
func ToHex(color Oklch) (string, error) {
	if !isFinite(color.L) || !isFinite(color.C) || !isFinite(color.H) {
		return "", fmt.Errorf("blobatar: ToHex: non-finite OKLCh component")
	}
	rgb := resolveColor(color)
	out := make([]byte, 0, 7)
	out = append(out, '#')
	for _, v := range rgb {
		if !isFinite(v) {
			return "", fmt.Errorf("blobatar: ToHex: non-finite channel after resolve")
		}
		s := 0.0
		if v <= 0.0031308 {
			s = 12.92 * v
		} else {
			s = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		if !isFinite(s) {
			return "", fmt.Errorf("blobatar: ToHex: non-finite transfer result")
		}
		n := int(math.Round(s * 255))
		if n < 0 {
			n = 0
		} else if n > 255 {
			n = 255
		}
		if n < 16 {
			out = append(out, '0')
		}
		out = strconv.AppendInt(out, int64(n), 16)
	}
	return string(out), nil
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// errBadHex is the shared malformed-hex message.
func errBadHex(hex string) error {
	return fmt.Errorf("blobatar: FromHex: want #RRGGBB, got %q", hex)
}

// validateHexRGB reports whether hex is exactly "#RRGGBB" / "#rrggbb"
// (leading '#' required; bare "rrggbb" is rejected — stricter than the TS
// oracle's parseInt(hex.slice(1), 16), which silently accepts malformed input).
func validateHexRGB(hex string) error {
	if len(hex) != 7 || hex[0] != '#' {
		return errBadHex(hex)
	}
	for i := 1; i < 7; i++ {
		if !isHexDigit(hex[i]) {
			return errBadHex(hex)
		}
	}
	return nil
}

// FromHex decodes a "#rrggbb" or "#RRGGBB" string into OKLCh.
//
// The leading '#' is required. Bare "rrggbb", empty string, a lone "#", wrong
// lengths, and non-hex characters all return an error — FromHex never panics
// and never invents a color for malformed input. The ParseInt error is not
// dropped.
func FromHex(hex string) (Oklch, error) {
	if err := validateHexRGB(hex); err != nil {
		return Oklch{}, err
	}
	n, err := strconv.ParseInt(hex[1:], 16, 64)
	if err != nil {
		return Oklch{}, fmt.Errorf("blobatar: FromHex: %w", err)
	}
	channels := [3]float64{
		float64((n >> 16) & 255),
		float64((n >> 8) & 255),
		float64(n & 255),
	}
	lin := func(v float64) float64 {
		s := v / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	r, g, b := lin(channels[0]), lin(channels[1]), lin(channels[2])

	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)

	A := 1.9779984951*l - 2.428592205*m + 0.4505937099*s
	B := 0.0259040371*l + 0.7827717662*m - 0.808675766*s

	return Oklch{
		L: 0.2104542553*l + 0.793617785*m - 0.0040720468*s,
		C: math.Hypot(A, B),
		H: (math.Atan2(B, A) * 180) / math.Pi,
	}, nil
}

// Mix blends a and b in OKLab at t in [0, 1].
func Mix(a, b Oklch, t float64) Oklch {
	rad := func(v float64) float64 { return (v * math.Pi) / 180 }
	ax := a.C * math.Cos(rad(a.H))
	ay := a.C * math.Sin(rad(a.H))
	bx := b.C * math.Cos(rad(b.H))
	by := b.C * math.Sin(rad(b.H))
	x := ax + (bx-ax)*t
	y := ay + (by-ay)*t
	return Oklch{
		L: a.L + (b.L-a.L)*t,
		C: math.Hypot(x, y),
		H: (math.Atan2(y, x) * 180) / math.Pi,
	}
}

// MixHex blends two serialized colors in OKLab and returns lowercase #rrggbb.
// Both a and b must be valid #RRGGBB (see FromHex); FromHex/ToHex errors are
// propagated.
func MixHex(a, b string, t float64) (string, error) {
	ca, err := FromHex(a)
	if err != nil {
		return "", err
	}
	cb, err := FromHex(b)
	if err != nil {
		return "", err
	}
	return ToHex(Mix(ca, cb, t))
}

// FadeHex blends two hex colors in sRGB (matching CSS transition: fill).
// Both a and b must be valid #RRGGBB (see FromHex); errors are propagated.
func FadeHex(a, b string, t float64) (string, error) {
	if err := validateHexRGB(a); err != nil {
		return "", err
	}
	if err := validateHexRGB(b); err != nil {
		return "", err
	}
	out := make([]byte, 0, 7)
	out = append(out, '#')
	for i := 1; i < 7; i += 2 {
		from, err := strconv.ParseInt(a[i:i+2], 16, 0)
		if err != nil {
			return "", fmt.Errorf("blobatar: FadeHex: %w", err)
		}
		to, err := strconv.ParseInt(b[i:i+2], 16, 0)
		if err != nil {
			return "", fmt.Errorf("blobatar: FadeHex: %w", err)
		}
		v := int(math.Round(float64(from) + (float64(to)-float64(from))*t))
		if v < 16 {
			out = append(out, '0')
		}
		out = strconv.AppendInt(out, int64(v), 16)
	}
	return string(out), nil
}

// Tint describes where a tinting pose is heading.
type Tint struct {
	H    float64
	L    float64
	Pull float64
	C    float64
}

// NamedTint pairs a tint target with its suite name.
type NamedTint struct {
	Name string
	Tint Tint
}

// Authored tint targets from vendor/core/color.ts (unexported; use accessors).
var (
	hot   = Tint{H: 27, L: 0.58, Pull: 0.6, C: 0.18}
	rose  = Tint{H: 358, L: 0.72, Pull: 0.55, C: 0.16}
	blush = Tint{H: 12, L: 0.84, Pull: 0.4, C: 0.1}
	bile  = Tint{H: 142, L: 0.66, Pull: 0.6, C: 0.13}
)

var tints = []NamedTint{
	{"hot", hot},
	{"rose", rose},
	{"blush", blush},
	{"bile", bile},
}

// HotTint returns a copy of the "hot" tint target.
func HotTint() Tint { return hot }

// RoseTint returns a copy of the "rose" tint target.
func RoseTint() Tint { return rose }

// BlushTint returns a copy of the "blush" tint target.
func BlushTint() Tint { return blush }

// BileTint returns a copy of the "bile" tint target.
func BileTint() Tint { return bile }

// TintTargets returns a copy of every tint the suite holds contrast for.
func TintTargets() []NamedTint {
	out := make([]NamedTint, len(tints))
	copy(out, tints)
	return out
}

const tintFloor = 4.55

var (
	darkSurface  = Oklch{L: 0.145, C: 0, H: 0}
	surfaceFloor = 1.5
)

// Tinted returns the palette a tinting pose heads toward.
// head and eye must be valid #RRGGBB (see FromHex); errors are propagated.
func Tinted(head, eye string, t Tint) (hotHead, hotEye string, err error) {
	base, err := FromHex(head)
	if err != nil {
		return "", "", err
	}
	baseEye, err := FromHex(eye)
	if err != nil {
		return "", "", err
	}

	hh := Oklch{
		L: base.L + (t.L-base.L)*t.Pull,
		C: math.Max(base.C, t.C),
		H: t.H,
	}
	hh = EnsureContrast(hh, darkSurface, surfaceFloor)

	he := EnsureContrast(baseEye, hh, tintFloor)

	dir := 1.0
	if he.L < hh.L {
		dir = -1
	}
	headHex, err := ToHex(hh)
	if err != nil {
		return "", "", err
	}
	for pass := 0; pass < 40; pass++ {
		eyeHex, err := ToHex(he)
		if err != nil {
			return "", "", err
		}
		worst := math.Inf(1)
		for i := 0; i <= 10; i++ {
			tt := float64(i) / 10
			mixedEye, err := MixHex(eye, eyeHex, tt)
			if err != nil {
				return "", "", err
			}
			mixedHead, err := MixHex(head, headHex, tt)
			if err != nil {
				return "", "", err
			}
			ce, err := FromHex(mixedEye)
			if err != nil {
				return "", "", err
			}
			ch, err := FromHex(mixedHead)
			if err != nil {
				return "", "", err
			}
			c := Contrast(ce, ch)
			if c < worst {
				worst = c
			}
		}
		if worst >= tintFloor {
			return headHex, eyeHex, nil
		}
		l := math.Min(1, math.Max(0, he.L+dir*0.02))
		if l == he.L {
			return headHex, eyeHex, nil
		}
		he.L = l
	}
	eyeOut, err := ToHex(he)
	if err != nil {
		return "", "", err
	}
	return headHex, eyeOut, nil
}

// toneSwatch is one entry of the authored tone set.
type toneSwatch struct {
	edge float64
	l, c float64
}

var tones = []toneSwatch{
	{0.2, 0.86, 0.085},  // pastel
	{0.36, 0.9, 0.028},  // pale neutral
	{0.62, 0.73, 0.135}, // mid
	{0.8, 0.62, 0.165},  // deep
	{0.93, 0.87, 0.16},  // bright
	{1.0, 0.34, 0.035},  // ink
}

func toneAt(v float64) (l, c float64) {
	for _, t := range tones {
		if v < t.edge {
			return t.l, t.c
		}
	}
	return tones[0].l, tones[0].c
}

// Floor is a minimum contrast pair as [foreground, background, ratio].
type Floor struct {
	FG, BG string
	Min    float64
}

// floors is the authored contrast chain from vendor/core/color.ts.
var floors = []Floor{
	{"head", "bg", 1.25},
	{"eye", "head", 4.5},
}

// ContrastFloors returns a copy of the authored contrast floor chain.
func ContrastFloors() []Floor {
	out := make([]Floor, len(floors))
	copy(out, floors)
	return out
}

func authoredRamp(h, tone float64) map[string]Oklch {
	tl, tc := toneAt(tone)
	head := EnsureContrast(Oklch{L: tl, C: tc, H: h}, darkSurface, surfaceFloor)
	eye := Oklch{L: 0.97, C: 0.012, H: h}
	if head.L >= 0.5 {
		eye = Oklch{L: 0.17, C: 0.02, H: h}
	}
	return map[string]Oklch{
		"bg":   {L: 0.965, C: 0.01, H: h},
		"head": head,
		"eye":  eye,
	}
}

// Ramp returns the palette in OKLCh before hex encoding.
func Ramp(hue float64, enforce bool, tone float64) map[string]Oklch {
	r := authoredRamp(hue, tone)
	if enforce {
		for _, f := range floors {
			r[f.FG] = EnsureContrast(r[f.FG], r[f.BG], f.Min)
		}
	}
	return r
}

// PaletteColors returns the hex palette for hue/tone (lowercase #rrggbb).
// Named PaletteColors rather than Palette to leave the Palette type free.
// Returns an error if any slot's ToHex fails (e.g. non-finite hue/tone).
func PaletteColors(hue float64, enforce bool, tone float64) (Palette, error) {
	r := Ramp(hue, enforce, tone)
	bg, err := ToHex(r["bg"])
	if err != nil {
		return Palette{}, err
	}
	head, err := ToHex(r["head"])
	if err != nil {
		return Palette{}, err
	}
	eye, err := ToHex(r["eye"])
	if err != nil {
		return Palette{}, err
	}
	return Palette{BG: bg, Head: head, Eye: eye}, nil
}
