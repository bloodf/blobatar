package blobatar_test

import (
	"math"
	"testing"

	"github.com/bloodf/blobatar"
	"github.com/bloodf/blobatar/internal/golden"
)

const floatEps = 1e-9

// closeEnough reports whether got matches want within eps.
// NaN never matches (including NaN vs NaN), so a NaN regression fails loudly.
func closeEnough(got, want, eps float64) bool {
	if math.IsNaN(got) || math.IsNaN(want) {
		return false
	}
	return math.Abs(got-want) <= eps
}

// Skip-gated parity suites. Each remaining skip names the groundwork item
// that will enable it (see bloodf/beans#379).

func TestTraitsPaletteParity(t *testing.T) {
	cases, err := golden.LoadTraits()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1000 {
		t.Fatalf("traits.jsonl: got %d rows want 1000", len(cases))
	}
	for _, tc := range cases {
		tr := blobatar.Traits(tc.Seed)
		gotHue := tr.Num("hue", 0, 360)
		gotTone := tr.Float("tone")
		gotName := blobatar.ToneName(gotTone)
		gotPal, err := blobatar.PaletteColors(gotHue, true, gotTone)
		if err != nil {
			t.Errorf("%q PaletteColors: %v", tc.Seed, err)
			continue
		}

		if !closeEnough(gotHue, tc.Hue, floatEps) {
			t.Errorf("%q hue: got %v want %v (Δ=%g)", tc.Seed, gotHue, tc.Hue, math.Abs(gotHue-tc.Hue))
		}
		if !closeEnough(gotTone, tc.Tone, floatEps) {
			t.Errorf("%q tone: got %v want %v (Δ=%g)", tc.Seed, gotTone, tc.Tone, math.Abs(gotTone-tc.Tone))
		}
		if gotName != tc.ToneName {
			t.Errorf("%q toneName: got %q want %q", tc.Seed, gotName, tc.ToneName)
		}
		if gotPal.BG != tc.Palette.BG || gotPal.Head != tc.Palette.Head || gotPal.Eye != tc.Palette.Eye {
			t.Errorf("%q palette: got {%s %s %s} want {%s %s %s}",
				tc.Seed, gotPal.BG, gotPal.Head, gotPal.Eye,
				tc.Palette.BG, tc.Palette.Head, tc.Palette.Eye)
		}

		// SeedPalette must agree with the same oracle fields.
		h2, t2, n2, p2, err := blobatar.SeedPalette(tc.Seed)
		if err != nil {
			t.Errorf("%q SeedPalette: %v", tc.Seed, err)
			continue
		}
		if !closeEnough(h2, tc.Hue, floatEps) || !closeEnough(t2, tc.Tone, floatEps) || n2 != tc.ToneName ||
			p2.BG != tc.Palette.BG || p2.Head != tc.Palette.Head || p2.Eye != tc.Palette.Eye {
			t.Errorf("%q SeedPalette mismatch (hue=%v tone=%v name=%q pal=%+v)", tc.Seed, h2, t2, n2, p2)
		}
	}
}

func TestShapeGeometryParity(t *testing.T) {
	t.Skip("blobatar epic G3: shape geometry (bloodf/beans#379)")
	// TODO(G3): compare Geometry() to geometry.jsonl within 1e-9.
}

func TestExpressionsLookParity(t *testing.T) {
	t.Skip("blobatar epic G4b: expressions+eyes (bloodf/beans#379)")
	// TODO(G4b): 14 expressions and every state vs geometry fixtures within 1e-9.
	// ValidateLook / ResolveAppearance / looks.json are covered by TestLook* (G4a).
}

func TestSVGRenderParity(t *testing.T) {
	t.Skip("blobatar epic G5: SVG render (bloodf/beans#379)")
	g, err := golden.LoadGen2()
	if err != nil {
		t.Fatal(err)
	}
	_ = g
	// TODO(G5): SVG() byte-exact against every gen2.txt markup line.
}

func TestMorphFrameParity(t *testing.T) {
	t.Skip("blobatar epic G6: morph+frame (bloodf/beans#379)")
	// TODO(G6): Lerp and Frame match frames.jsonl within 1e-9.
}
