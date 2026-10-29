package blobatar_test

import (
	"math"
	"testing"

	"github.com/bloodf/blobatar"
	"github.com/bloodf/blobatar/internal/golden"
)

const floatEps = 1e-9

// Skip-gated parity suites. Each remaining skip names the groundwork item
// that will enable it (see bloodf/beans#379).

func TestTraitsPaletteParity(t *testing.T) {
	cases, err := golden.LoadTraits()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("traits.jsonl is empty")
	}
	for _, tc := range cases {
		tr := blobatar.Traits(tc.Seed)
		gotHue := tr.Num("hue", 0, 360)
		gotTone := tr.Float("tone")
		gotName := blobatar.ToneName(gotTone)
		gotPal := blobatar.PaletteColors(gotHue, true, gotTone)

		if math.Abs(gotHue-tc.Hue) > floatEps {
			t.Errorf("%q hue: got %v want %v (Δ=%g)", tc.Seed, gotHue, tc.Hue, math.Abs(gotHue-tc.Hue))
		}
		if math.Abs(gotTone-tc.Tone) > floatEps {
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
		h2, t2, n2, p2 := blobatar.SeedPalette(tc.Seed)
		if math.Abs(h2-tc.Hue) > floatEps || math.Abs(t2-tc.Tone) > floatEps || n2 != tc.ToneName ||
			p2.BG != tc.Palette.BG || p2.Head != tc.Palette.Head || p2.Eye != tc.Palette.Eye {
			t.Errorf("%q SeedPalette mismatch", tc.Seed)
		}
	}
}

func TestShapeGeometryParity(t *testing.T) {
	t.Skip("blocked on groundwork G3: shape geometry")
	// TODO(groundwork G3): compare Geometry() to geometry.jsonl within 1e-9.
}

func TestExpressionsLookParity(t *testing.T) {
	t.Skip("blocked on groundwork G4b: expressions+look")
	// TODO(groundwork G4b): ValidateLook / ResolveAppearance / 14 expressions.
}

func TestSVGRenderParity(t *testing.T) {
	t.Skip("blocked on groundwork G5: SVG render")
	g, err := golden.LoadGen2()
	if err != nil {
		t.Fatal(err)
	}
	_ = g
	// TODO(groundwork G5): SVG() byte-exact against every gen2.txt markup line.
}

func TestMorphFrameParity(t *testing.T) {
	t.Skip("blocked on groundwork G6: morph+frame")
	// TODO(groundwork G6): Lerp and Frame match frames.jsonl within 1e-9.
}
