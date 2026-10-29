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

func TestColorOracleParity(t *testing.T) {
	cases, err := golden.LoadColorOracle()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("color_oracle.jsonl: empty")
	}

	tintOf := map[string]blobatar.Tint{
		"hot":   blobatar.HotTint(),
		"rose":  blobatar.RoseTint(),
		"blush": blobatar.BlushTint(),
		"bile":  blobatar.BileTint(),
	}

	for i, tc := range cases {
		switch {
		case tc.Error == "bad_t":
			if tc.Fn != "fadeHex" {
				t.Fatalf("%d: bad_t only valid for fadeHex, got %q", i, tc.Fn)
			}
			tv, err := tc.ParseT()
			if err != nil {
				t.Fatalf("%d: %v", i, err)
			}
			got, err := blobatar.FadeHex(tc.A, tc.B, tv)
			if err == nil {
				t.Errorf("%d: FadeHex bad_t=%v returned %q, want error", i, tv, got)
			} else if got != "" {
				t.Errorf("%d: FadeHex bad_t returned %q with error", i, got)
			}

		case tc.Error == "bad_hex":
			switch tc.Fn {
			case "mixHex":
				tv, err := tc.ParseT()
				if err != nil {
					t.Fatalf("%d: %v", i, err)
				}
				got, err := blobatar.MixHex(tc.A, tc.B, tv)
				if err == nil {
					t.Errorf("%d: MixHex(%q,%q) = %q, want error", i, tc.A, tc.B, got)
				}
			case "fadeHex":
				tv, err := tc.ParseT()
				if err != nil {
					t.Fatalf("%d: %v", i, err)
				}
				got, err := blobatar.FadeHex(tc.A, tc.B, tv)
				if err == nil {
					t.Errorf("%d: FadeHex(%q,%q) = %q, want error", i, tc.A, tc.B, got)
				}
			case "tinted":
				tint, ok := tintOf[tc.Tint]
				if !ok {
					t.Fatalf("%d: unknown tint %q", i, tc.Tint)
				}
				h, e, err := blobatar.Tinted(tc.Head, tc.Eye, tint)
				if err == nil {
					t.Errorf("%d: Tinted(%q,%q) = %q,%q, want error", i, tc.Head, tc.Eye, h, e)
				}
			default:
				t.Fatalf("%d: bad_hex for unknown fn %q", i, tc.Fn)
			}

		case tc.Fn == "mixHex":
			tv, err := tc.ParseT()
			if err != nil {
				t.Fatalf("%d: %v", i, err)
			}
			got, err := blobatar.MixHex(tc.A, tc.B, tv)
			if err != nil {
				t.Errorf("%d: MixHex: %v", i, err)
				continue
			}
			if got != tc.Hex {
				t.Errorf("%d: MixHex(%q,%q,%v)=%q want %q", i, tc.A, tc.B, tv, got, tc.Hex)
			}

		case tc.Fn == "fadeHex":
			tv, err := tc.ParseT()
			if err != nil {
				t.Fatalf("%d: %v", i, err)
			}
			got, err := blobatar.FadeHex(tc.A, tc.B, tv)
			if err != nil {
				t.Errorf("%d: FadeHex: %v", i, err)
				continue
			}
			if got != tc.Hex {
				t.Errorf("%d: FadeHex(%q,%q,%v)=%q want %q", i, tc.A, tc.B, tv, got, tc.Hex)
			}

		case tc.Fn == "tinted":
			tint, ok := tintOf[tc.Tint]
			if !ok {
				t.Fatalf("%d: unknown tint %q", i, tc.Tint)
			}
			headOut, eyeOut, err := blobatar.Tinted(tc.Head, tc.Eye, tint)
			if err != nil {
				t.Errorf("%d: Tinted seed=%q tint=%q: %v", i, tc.Seed, tc.Tint, err)
				continue
			}
			if headOut != tc.HeadOut || eyeOut != tc.EyeOut {
				t.Errorf("%d: Tinted seed=%q tint=%q: got (%q,%q) want (%q,%q)",
					i, tc.Seed, tc.Tint, headOut, eyeOut, tc.HeadOut, tc.EyeOut)
			}

		default:
			t.Fatalf("%d: unhandled color oracle row fn=%q error=%q", i, tc.Fn, tc.Error)
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
