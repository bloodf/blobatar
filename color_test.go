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

	upper, err := blobatar.FromHex("#AABBCC")
	if err != nil {
		t.Fatal(err)
	}
	if upper != c {
		t.Fatalf("uppercase FromHex mismatch: %+v vs %+v", upper, c)
	}

	r := blobatar.Ramp(210, true, 0.5)
	hex, err := blobatar.ToHex(r["head"])
	if err != nil {
		t.Fatal(err)
	}
	if hex != "#00bcd3" {
		t.Fatalf("ToHex(ramp 210 mid head)=%q want #00bcd3", hex)
	}
	ratio := blobatar.Contrast(r["eye"], r["head"])
	if !almostEqual(ratio, 8.330782754251093, 1e-9) {
		t.Fatalf("eye/head contrast=%v", ratio)
	}
}

func TestFromHexMalformed(t *testing.T) {
	cases := []struct {
		hex string
	}{
		{""},
		{"#"},
		{"#12"}, // 2 hex digits after #
		{"#zz"},
		{"#11223344"}, // 8 hex digits after #
		{"abc"},
		{"aabbcc"}, // bare rrggbb — leading '#' required
		{"#abc"},
		{"#abcd"},
		{"#abcdef0"},
		{"#gghhii"},
		{"#aabbgg"},
		{"#AABB C"},
		{"##abbcc"},
		{"#aabbcc "},
		{" #aabbcc"},
	}
	for _, tc := range cases {
		c, err := blobatar.FromHex(tc.hex)
		if err == nil {
			t.Errorf("FromHex(%q) = %+v, want error", tc.hex, c)
		}
	}
}

func TestHexHelpersPropagateFromHexError(t *testing.T) {
	// FadeHex/MixHex/Tinted must not panic on short/empty strings (no raw slice).
	bad := []string{"", "#", "#12", "#zz", "#11223344", "abc", "aabbcc"}
	for _, hex := range bad {
		if _, err := blobatar.MixHex(hex, "#000000", 0.5); err == nil {
			t.Errorf("MixHex(%q, …) want error", hex)
		}
		if _, err := blobatar.MixHex("#000000", hex, 0.5); err == nil {
			t.Errorf("MixHex(…, %q) want error", hex)
		}
		if _, err := blobatar.FadeHex(hex, "#ffffff", 0.5); err == nil {
			t.Errorf("FadeHex(%q, …) want error", hex)
		}
		if _, err := blobatar.FadeHex("#ffffff", hex, 0.5); err == nil {
			t.Errorf("FadeHex(…, %q) want error", hex)
		}
		if _, _, err := blobatar.Tinted(hex, "#000000", blobatar.HotTint()); err == nil {
			t.Errorf("Tinted(%q, …) want error", hex)
		}
		if _, _, err := blobatar.Tinted("#ffffff", hex, blobatar.HotTint()); err == nil {
			t.Errorf("Tinted(…, %q) want error", hex)
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

func TestToHexNonFiniteErrors(t *testing.T) {
	// Documented behavior: ToHex returns an error for NaN / ±Inf (never
	// "#000000", never platform-dependent int(NaN)).
	cases := []blobatar.Oklch{
		{L: math.NaN(), C: 0, H: 0},
		{L: math.Inf(1), C: 0, H: 0},
		{L: math.Inf(-1), C: 0, H: 0},
		{L: 0.5, C: math.NaN(), H: 0},
		{L: 0.5, C: 0.1, H: math.Inf(1)},
		{L: math.NaN(), C: math.NaN(), H: math.NaN()},
	}
	for _, c := range cases {
		got, err := blobatar.ToHex(c)
		if err == nil {
			t.Fatalf("ToHex(%+v) = %q, want error", c, got)
		}
		if got != "" {
			t.Fatalf("ToHex(%+v) returned %q with error %v", c, got, err)
		}
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
	p, err := blobatar.PaletteColors(hue, true, tone)
	if err != nil {
		t.Fatal(err)
	}
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
		for _, f := range blobatar.ContrastFloors() {
			got := blobatar.Contrast(r[f.FG], r[f.BG])
			if got < f.Min-1e-9 {
				t.Fatalf("hue %d %s/%s contrast %v < %v", h, f.FG, f.BG, got, f.Min)
			}
		}
		p, err := blobatar.PaletteColors(float64(h), true, 0)
		if err != nil {
			t.Fatalf("hue %d: %v", h, err)
		}
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
	same, err := blobatar.FadeHex(a, a, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if same != a {
		t.Fatalf("FadeHex same: %q", same)
	}
	mid, err := blobatar.FadeHex("#000000", "#ffffff", 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if mid != "#808080" {
		t.Fatalf("FadeHex mid: %q want #808080", mid)
	}
	// Endpoints of the allowed t range.
	for _, tVal := range []float64{0, 1} {
		if _, err := blobatar.FadeHex("#000000", "#ffffff", tVal); err != nil {
			t.Fatalf("FadeHex t=%v: %v", tVal, err)
		}
	}
}

func TestFadeHexRejectsBadT(t *testing.T) {
	// NaN / ±Inf / outside [0,1] must error — never malformed hex like the
	// TS oracle's "#0-ff0-ff0-ff" (t=-1) or "#NaNNaNNaN" (t=NaN).
	bad := []float64{
		math.NaN(),
		math.Inf(1),
		math.Inf(-1),
		-1,
		-0.0001,
		1.0001,
		2,
		math.MaxFloat64,
	}
	for _, tVal := range bad {
		got, err := blobatar.FadeHex("#000000", "#ffffff", tVal)
		if err == nil {
			t.Errorf("FadeHex(t=%v) = %q, want error", tVal, got)
			continue
		}
		if got != "" {
			t.Errorf("FadeHex(t=%v) returned %q with error %v", tVal, got, err)
		}
		// Must never look like a hex string with '-', 'N', or 3+ digit channels.
		if len(got) != 0 && (len(got) != 7 || got[0] != '#') {
			t.Errorf("FadeHex(t=%v) leaked malformed %q", tVal, got)
		}
	}
}

func TestTintedHeatZero(t *testing.T) {
	p, err := blobatar.PaletteColors(200, true, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	hotHead, hotEye, err := blobatar.Tinted(p.Head, p.Eye, blobatar.HotTint())
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

func TestAccessorsReturnCopies(t *testing.T) {
	edges := blobatar.ToneBandEdges()
	edges[0] = -1
	if blobatar.ToneBandEdges()[0] != 0.2 {
		t.Fatal("ToneBandEdges must return a copy")
	}
	names := blobatar.ToneBandNames()
	names[0] = "x"
	if blobatar.ToneBandNames()[0] != "pastel" {
		t.Fatal("ToneBandNames must return a copy")
	}
	floors := blobatar.ContrastFloors()
	floors[0].Min = 99
	if blobatar.ContrastFloors()[0].Min != 1.25 {
		t.Fatal("ContrastFloors must return a copy")
	}
	tints := blobatar.TintTargets()
	tints[0].Tint.H = -1
	if blobatar.TintTargets()[0].Tint.H != blobatar.HotTint().H {
		t.Fatal("TintTargets must return a copy")
	}
}
