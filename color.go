// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/color.ts), MIT License.
// See NOTICE for the full license text.

package blobatar

import (
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

func inGamut(rgb [3]float64) bool {
	for _, v := range rgb {
		if v < -1e-4 || v > 1+1e-4 {
			return false
		}
	}
	return true
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
func ToHex(color Oklch) string {
	rgb := resolveColor(color)
	out := make([]byte, 0, 7)
	out = append(out, '#')
	for _, v := range rgb {
		s := 0.0
		if v <= 0.0031308 {
			s = 12.92 * v
		} else {
			s = 1.055*math.Pow(v, 1/2.4) - 0.055
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
	return string(out)
}

// FromHex decodes an #rrggbb (or #RRGGBB) string into OKLCh.
func FromHex(hex string) Oklch {
	n, _ := strconv.ParseInt(hex[1:], 16, 64)
	channels := [3]float64{
		float64((n >> 16) & 255),
		float64((n >> 8) & 255),
		float64(n & 255),
	}
	var r, g, b float64
	lin := func(v float64) float64 {
		s := v / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	r, g, b = lin(channels[0]), lin(channels[1]), lin(channels[2])

	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)

	A := 1.9779984951*l - 2.428592205*m + 0.4505937099*s
	B := 0.0259040371*l + 0.7827717662*m - 0.808675766*s

	return Oklch{
		L: 0.2104542553*l + 0.793617785*m - 0.0040720468*s,
		C: math.Hypot(A, B),
		H: (math.Atan2(B, A) * 180) / math.Pi,
	}
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
func MixHex(a, b string, t float64) string {
	return ToHex(Mix(FromHex(a), FromHex(b), t))
}

// FadeHex blends two hex colors in sRGB (matching CSS transition: fill).
func FadeHex(a, b string, t float64) string {
	out := make([]byte, 0, 7)
	out = append(out, '#')
	for i := 1; i < 7; i += 2 {
		from, _ := strconv.ParseInt(a[i:i+2], 16, 0)
		to, _ := strconv.ParseInt(b[i:i+2], 16, 0)
		v := int(math.Round(float64(from) + (float64(to)-float64(from))*t))
		if v < 16 {
			out = append(out, '0')
		}
		out = strconv.AppendInt(out, int64(v), 16)
	}
	return string(out)
}

// Tint describes where a tinting pose is heading.
type Tint struct {
	H    float64
	L    float64
	Pull float64
	C    float64
}

// Authored tint targets from vendor/core/color.ts.
var (
	HOT   = Tint{H: 27, L: 0.58, Pull: 0.6, C: 0.18}
	ROSE  = Tint{H: 358, L: 0.72, Pull: 0.55, C: 0.16}
	BLUSH = Tint{H: 12, L: 0.84, Pull: 0.4, C: 0.1}
	BILE  = Tint{H: 142, L: 0.66, Pull: 0.6, C: 0.13}
)

// Tints is every target the suite holds the contrast guarantee across.
var Tints = []struct {
	Name string
	Tint Tint
}{
	{"hot", HOT},
	{"rose", ROSE},
	{"blush", BLUSH},
	{"bile", BILE},
}

const tintFloor = 4.55

var (
	darkSurface  = Oklch{L: 0.145, C: 0, H: 0}
	surfaceFloor = 1.5
)

// Tinted returns the palette a tinting pose heads toward.
func Tinted(head, eye string, t Tint) (hotHead, hotEye string) {
	base := FromHex(head)
	baseEye := FromHex(eye)

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
	headHex := ToHex(hh)
	for pass := 0; pass < 40; pass++ {
		eyeHex := ToHex(he)
		worst := math.Inf(1)
		for i := 0; i <= 10; i++ {
			tt := float64(i) / 10
			c := Contrast(FromHex(MixHex(eye, eyeHex, tt)), FromHex(MixHex(head, headHex, tt)))
			if c < worst {
				worst = c
			}
		}
		if worst >= tintFloor {
			return headHex, eyeHex
		}
		l := math.Min(1, math.Max(0, he.L+dir*0.02))
		if l == he.L {
			return headHex, eyeHex
		}
		he.L = l
	}
	return headHex, ToHex(he)
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

// Floors is the authored contrast chain from vendor/core/color.ts.
var Floors = []Floor{
	{"head", "bg", 1.25},
	{"eye", "head", 4.5},
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
		for _, f := range Floors {
			r[f.FG] = EnsureContrast(r[f.FG], r[f.BG], f.Min)
		}
	}
	return r
}

// PaletteColors returns the hex palette for hue/tone (lowercase #rrggbb).
// Named PaletteColors rather than Palette to leave the Palette type free.
func PaletteColors(hue float64, enforce bool, tone float64) Palette {
	r := Ramp(hue, enforce, tone)
	return Palette{
		BG:   ToHex(r["bg"]),
		Head: ToHex(r["head"]),
		Eye:  ToHex(r["eye"]),
	}
}
