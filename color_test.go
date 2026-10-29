package blobatar_test

import (
	"math"
	"testing"

	"github.com/bloodf/blobatar"
)

func almostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

func TestContrastEnsureContrast(t *testing.T) {
	bg := blobatar.Oklch{L: 0.6, C: 0.1, H: 240}
	fg := blobatar.Oklch{L: 0.62, C: 0.1, H: 240}
	if blobatar.Contrast(fg, bg) >= 1.2 {
		t.Fatalf("pair should start too close")
	}
	fixed := blobatar.EnsureContrast(fg, bg, 4.5)
	if blobatar.Contrast(fixed, bg) < 4.5-1e-9 {
		t.Fatalf("ensureContrast failed: ratio=%v", blobatar.Contrast(fixed, bg))
	}
	// TS oracle for this pair lands near L≈0.2.
	if !almostEqual(fixed.L, 0.2, 1e-9) {
		t.Fatalf("ensureContrast L: got %v want ~0.2", fixed.L)
	}

	lightBG := blobatar.Oklch{L: 0.9, C: 0.05, H: 30}
	leanFG := blobatar.Oklch{L: 0.85, C: 0.05, H: 30}
	got := blobatar.EnsureContrast(leanFG, lightBG, 4.5)
	if got.L >= leanFG.L {
		t.Fatalf("should lean darker, got L=%v", got.L)
	}

	passBG := blobatar.Oklch{L: 0.95, C: 0.02, H: 30}
	passFG := blobatar.Oklch{L: 0.1, C: 0.02, H: 30}
	untouched := blobatar.EnsureContrast(passFG, passBG, 4.5)
	if untouched != passFG {
		t.Fatalf("already-passing pair must be unchanged")
	}
}

func TestToHexFromHexRoundTripKnown(t *testing.T) {
	c, err := blobatar.FromHex("#aabbcc")
	if err != nil {
		t.Fatal(err)
	}
	// Values observed from Beans vendor/core/color.ts under bun 1.4.4.
	if !almostEqual(c.L, 0.784436469100708, 1e-9) ||
		!almostEqual(c.C, 0.030668810150649508, 1e-9) ||
		!almostEqual(c.H, -111.7818027453672, 1e-9) {
		t.Fatalf("FromHex(#aabbcc)=%+v", c)
	}

	r := blobatar.Ramp(210, true, 0.5)
	hex := blobatar.ToHex(r["head"])
	if hex != "#00bcd3" {
		t.Fatalf("ToHex(ramp 210 mid head)=%q want #00bcd3", hex)
	}
	ratio := blobatar.Contrast(r["eye"], r["head"])
	if !almostEqual(ratio, 8.330782754251093, 1e-9) {
		t.Fatalf("eye/head contrast=%v", ratio)
	}
}

func TestFromHexMalformed(t *testing.T) {
	cases := []string{
		"",
		"#",
		"aabbcc",
		"#abc",
		"#abcd",
		"#abcdef0",
		"#gghhii",
		"#aabbgg",
		"#AABB C",
		"##abbcc",
		"#aabbcc ",
		" #aabbcc",
	}
	for _, hex := range cases {
		c, err := blobatar.FromHex(hex)
		if err == nil {
			t.Errorf("FromHex(%q) = %+v, want error", hex, c)
		}
	}
}

func TestInGamutNaNInf(t *testing.T) {
	if !blobatar.InGamutForTest(0, 0.5, 1) {
		t.Fatal("in-range RGB must be in gamut")
	}
	if blobatar.InGamutForTest(math.NaN(), 0, 0) {
		t.Fatal("NaN must be out of gamut")
	}
	if blobatar.InGamutForTest(0, math.NaN(), 0) {
		t.Fatal("NaN (g) must be out of gamut")
	}
	if blobatar.InGamutForTest(0, 0, math.NaN()) {
		t.Fatal("NaN (b) must be out of gamut")
	}
	if blobatar.InGamutForTest(math.Inf(1), 0, 0) {
		t.Fatal("+Inf must be out of gamut")
	}
	if blobatar.InGamutForTest(math.Inf(-1), 0.5, 0.5) {
		t.Fatal("-Inf must be out of gamut")
	}
	if blobatar.InGamutForTest(-0.1, 0, 0) {
		t.Fatal("below lo must be out of gamut")
	}
	if blobatar.InGamutForTest(1.1, 0, 0) {
		t.Fatal("above hi must be out of gamut")
	}
}

func TestCloseEnoughRejectsNaN(t *testing.T) {
	// Mirrors the parity helper: NaN must not compare equal.
	if closeEnough(math.NaN(), 0, floatEps) {
		t.Fatal("NaN vs 0 must fail")
	}
	if closeEnough(0, math.NaN(), floatEps) {
		t.Fatal("0 vs NaN must fail")
	}
	if closeEnough(math.NaN(), math.NaN(), floatEps) {
		t.Fatal("NaN vs NaN must fail")
	}
}

func TestPaletteAlain(t *testing.T) {
	// Cross-check against bun oracle for seed "alain".
	hue := blobatar.Hue("alain")
	tone := blobatar.Tone("alain")
	if !almostEqual(hue, 113.5819791443646, 1e-9) {
		t.Fatalf("hue=%v", hue)
	}
	if !almostEqual(tone, 0.1290204932447523, 1e-9) {
		t.Fatalf("tone=%v", tone)
	}
	p := blobatar.PaletteColors(hue, true, tone)
	if p.BG != "#f3f4ed" || p.Head != "#d0d897" || p.Eye != "#0f1006" {
		t.Fatalf("palette=%+v", p)
	}
	if blobatar.ToneName(tone) != "pastel" {
		t.Fatalf("toneName=%q", blobatar.ToneName(tone))
	}
}

func TestRampFloorsAcrossHues(t *testing.T) {
	for h := 0; h < 360; h++ {
		r := blobatar.Ramp(float64(h), true, 0)
		for _, f := range blobatar.Floors {
			got := blobatar.Contrast(r[f.FG], r[f.BG])
			if got < f.Min-1e-9 {
				t.Fatalf("hue %d %s/%s contrast %v < %v", h, f.FG, f.BG, got, f.Min)
			}
		}
		p := blobatar.PaletteColors(float64(h), true, 0)
		for _, hex := range []string{p.BG, p.Head, p.Eye} {
			if len(hex) != 7 || hex[0] != '#' {
				t.Fatalf("bad hex %q at hue %d", hex, h)
			}
		}
	}
}

func TestEyePolarityAcrossTones(t *testing.T) {
	tones := []float64{0.1, 0.3, 0.5, 0.7, 0.88, 0.97}
	for _, tone := range tones {
		r := blobatar.Ramp(200, true, tone)
		darkEye := r["eye"].L < 0.5
		lightHead := r["head"].L >= 0.5
		if darkEye != lightHead {
			t.Fatalf("tone %v polarity: eye.L=%v head.L=%v", tone, r["eye"].L, r["head"].L)
		}
		if blobatar.Contrast(r["eye"], r["head"]) < 4.5-1e-9 {
			t.Fatalf("tone %v eye/head under floor", tone)
		}
	}
}

func TestMixHexFadeHex(t *testing.T) {
	a, b := "#112233", "#aabbcc"
	got, err := blobatar.MixHex(a, b, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != a {
		t.Fatalf("MixHex t=0: %q", got)
	}
	if got := blobatar.FadeHex(a, a, 0.5); got != a {
		t.Fatalf("FadeHex same: %q", got)
	}
	mid := blobatar.FadeHex("#000000", "#ffffff", 0.5)
	if mid != "#808080" {
		t.Fatalf("FadeHex mid: %q want #808080", mid)
	}
}

func TestTintedHeatZero(t *testing.T) {
	p := blobatar.PaletteColors(200, true, 0.5)
	hotHead, hotEye, err := blobatar.Tinted(p.Head, p.Eye, blobatar.HOT)
	if err != nil {
		t.Fatal(err)
	}
	gotHead, err := blobatar.MixHex(p.Head, hotHead, 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotHead != p.Head {
		t.Fatalf("heat 0 head drifted")
	}
	gotEye, err := blobatar.MixHex(p.Eye, hotEye, 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotEye != p.Eye {
		t.Fatalf("heat 0 eye drifted")
	}
}
