package blobatar

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

type expressionCase struct {
	InputPalette *Palette          `json:"inputPalette"`
	Name         string            `json:"name"`
	Seed         string            `json:"seed"`
	Expression   string            `json:"expression"`
	Overrides    TraitOverrides    `json:"overrides"`
	Pose         pose              `json:"pose"`
	Vars         map[string]string `json:"vars"`
	Eyes         json.RawMessage   `json:"eyes"`
	Wrap         string            `json:"wrap"`
	EyePaths     [2]string         `json:"eyePaths"`
	Transforms   []struct {
		Phase    float64       `json:"phase"`
		Expected poseTransform `json:"expected"`
	} `json:"transforms"`
	Palette Palette `json:"palette"`
}

func checkExpressionCase(t *testing.T, c expressionCase, p pose) {
	t.Helper()
	l := composeLayout(NewTraits(c.Seed, true, c.Overrides))
	before, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	baked, wrap := bakePose(l, p)
	assertLayoutJSON(t, baked.Eyes, c.Eyes)
	if wrap != c.Wrap {
		t.Errorf("wrap: got %q, want %q", wrap, c.Wrap)
	}
	if got := poseVars(p); !reflect.DeepEqual(got, c.Vars) {
		t.Errorf("vars: got %v, want %v", got, c.Vars)
	}
	for i, eye := range baked.Eyes {
		if got := eye.path(); got != c.EyePaths[i] {
			t.Errorf("eye %d: got %s, want %s", i, got, c.EyePaths[i])
		}
	}
	for _, c := range c.Transforms {
		if got := poseTransforms(l, p, c.Phase); got != c.Expected {
			t.Errorf("phase %g transforms: got %+v, want %+v", c.Phase, got, c.Expected)
		}
	}
	after, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("bake or transforms changed the input layout")
	}
	// Baking must leave every non-eye field untouched in its returned layout.
	baked.Eyes = l.Eyes
	if !reflect.DeepEqual(baked, l) {
		t.Fatal("bake changed the silhouette or face region")
	}
}

func TestExpressionMatchesBeans(t *testing.T) {
	seen := map[string]int{}
	count := readLayoutRows(t, "testdata/expressions/expressions.jsonl", func(raw []byte) {
		var c expressionCase
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(t *testing.T) {
			e, err := expressionForName(c.Expression)
			if err != nil {
				t.Fatal(err)
			}
			if e.P != c.Pose {
				t.Fatalf("pose: got %+v, want %+v", e.P, c.Pose)
			}
			checkExpressionCase(t, c, e.P)
			_, _, _, base, err := SeedPalette(c.Seed)
			if err != nil {
				t.Fatal(err)
			}
			if c.InputPalette != nil {
				base = *c.InputPalette
			}
			got, err := expressionPalette(base, e)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.Palette {
				t.Errorf("palette: got %+v, want %+v", got, c.Palette)
			}
			seen[c.Expression]++
		})
	})
	if count != 1152 {
		t.Fatalf("expression oracle: got %d rows, want 1152", count)
	}
	for _, name := range expressions {
		if seen[name] == 0 {
			t.Errorf("missing expression %s", name)
		}
	}
	if len(expressionRoster) != len(expressions) {
		t.Fatal("expression roster differs from BotLook vocabulary")
	}
}

func TestCustomPoseMatchesBeans(t *testing.T) {
	count := readLayoutRows(t, "testdata/expressions/custom.jsonl", func(raw []byte) {
		var c expressionCase
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(t *testing.T) { checkExpressionCase(t, c, c.Pose) })
	})
	if count != 3 {
		t.Fatalf("custom pose oracle: got %d rows, want 3", count)
	}
}

func TestLerpPoseMatchesBeans(t *testing.T) {
	get := func(name *string) *pose {
		if name == nil {
			return nil
		}
		e, err := expressionForName(*name)
		if err != nil {
			t.Fatal(err)
		}
		return &e.P
	}
	count := readLayoutRows(t, "testdata/expressions/lerp.jsonl", func(raw []byte) {
		var c struct {
			From     *string `json:"from"`
			To       *string `json:"to"`
			T        float64 `json:"t"`
			Expected pose    `json:"expected"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		got := lerpPose(get(c.From), get(c.To), c.T)
		// These are weighted sums of literals, so check exact equality, including
		// the settled endpoints. Interpolation deliberately does not clamp t.
		if got != c.Expected {
			t.Fatalf("lerp t=%g: got %+v, want %+v", c.T, got, c.Expected)
		}
	})
	if count != 1350 {
		t.Fatalf("lerp pose oracle: got %d rows, want 1350", count)
	}
}

func TestPoseNumberMatchesBeans(t *testing.T) {
	count := readLayoutRows(t, "testdata/expressions/rounding.jsonl", func(raw []byte) {
		var c struct {
			Value    float64 `json:"value"`
			Expected string  `json:"expected"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		if got := poseNumber(c.Value); got != c.Expected {
			t.Errorf("r3(%g): got %q, want %q", c.Value, got, c.Expected)
		}
	})
	if count != 9 {
		t.Fatalf("rounding oracle: got %d rows, want 9", count)
	}
}

// Geometry carries unposed eyes and a separate pose. This checks pose and
// tinted palette against the original fixture; it does not claim that the
// production Geometry or static SVG assembly is implemented.
func TestExpressionGeometryGoldenSubset(t *testing.T) {
	row := 0
	count := readLayoutRows(t, "testdata/golden/geometry.jsonl", func(raw []byte) {
		var c struct {
			Seed     string `json:"seed"`
			Look     *Look  `json:"look"`
			State    string `json:"state"`
			Geometry struct {
				Pose    pose    `json:"pose"`
				Palette Palette `json:"palette"`
			} `json:"geometry"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		t.Run(fmt.Sprintf("row-%d", row), func(t *testing.T) {
			a, err := ResolveAppearance(c.Seed, c.Look, c.State)
			if err != nil {
				t.Fatal(err)
			}
			e, err := expressionForName(a.Expression)
			if err != nil {
				t.Fatal(err)
			}
			if e.P != c.Geometry.Pose {
				t.Errorf("pose: got %+v, want %+v", e.P, c.Geometry.Pose)
			}
			got, err := expressionPalette(Palette{BG: a.Palette.Bg, Head: a.Palette.Head, Eye: a.Palette.Eye}, e)
			if err != nil {
				t.Fatal(err)
			}
			got = Palette{BG: strings.ToUpper(got.BG), Head: strings.ToUpper(got.Head), Eye: strings.ToUpper(got.Eye)}
			if got != c.Geometry.Palette {
				t.Errorf("palette: got %+v, want %+v", got, c.Geometry.Palette)
			}
		})
		row++
	})
	if count != 1049 {
		t.Fatalf("geometry oracle: got %d rows, want 1049", count)
	}
}

func TestExpressionValueSemantics(t *testing.T) {
	e, err := expressionForName("")
	if err != nil || e.P != identityPose() {
		t.Fatalf("empty expression should be identity: %+v, %v", e, err)
	}
	e.P.ESX = 99
	again, err := expressionForName("idle")
	if err != nil || again.P != identityPose() {
		t.Fatal("expression lookup exposed mutable roster data")
	}
	if _, err := expressionForName("bogus"); err == nil {
		t.Fatal("accepted unknown expression")
	}
	pal := Palette{BG: "#ABCDEF", Head: "#123456", Eye: "#FFFFFF"}
	idle, _ := expressionForName("idle")
	if got, err := expressionPalette(pal, idle); err != nil || got != pal {
		t.Fatal("cool expression should preserve exact palette bytes")
	}
	if _, err := expressionPalette(pal, expression{tint: "bogus"}); err == nil {
		t.Fatal("accepted unknown tint")
	}
	mad, _ := expressionForName("mad")
	for _, bad := range []Palette{{Head: "bad", Eye: "#FFFFFF"}, {Head: "#123456", Eye: "bad"}} {
		if _, err := expressionPalette(bad, mad); err == nil {
			t.Fatal("tint did not propagate invalid palette error")
		}
	}
	mad.P.Heat = math.NaN()
	if _, err := expressionPalette(pal, mad); err == nil {
		t.Fatal("tint did not propagate non-finite heat error")
	}
}

// Every BotAvatarState can override each of the fourteen expressions. Exercise
// resolution plus tinting directly until full Geometry assembly is available.
func TestExpressionStateOverridesMatchesBeans(t *testing.T) {
	count := readLayoutRows(t, "testdata/expressions/states.jsonl", func(raw []byte) {
		var c struct {
			Seed    string  `json:"seed"`
			State   string  `json:"state"`
			Look    Look    `json:"look"`
			Pose    pose    `json:"pose"`
			Palette Palette `json:"palette"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		a, err := ResolveAppearance(c.Seed, c.Look, c.State)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.State+"/"+a.Expression, func(t *testing.T) {
			e, err := expressionForName(a.Expression)
			if err != nil {
				t.Fatal(err)
			}
			if e.P != c.Pose {
				t.Errorf("pose: got %+v, want %+v", e.P, c.Pose)
			}
			got, err := expressionPalette(Palette{BG: a.Palette.Bg, Head: a.Palette.Head, Eye: a.Palette.Eye}, e)
			if err != nil {
				t.Fatal(err)
			}
			got = Palette{BG: strings.ToUpper(got.BG), Head: strings.ToUpper(got.Head), Eye: strings.ToUpper(got.Eye)}
			if got != c.Palette {
				t.Errorf("palette: got %+v, want %+v", got, c.Palette)
			}
		})
	})
	if count != 98 {
		t.Fatalf("state expression oracle: got %d rows, want 98", count)
	}
}

func TestPoseSignedZero(t *testing.T) {
	zero := math.Copysign(0, -1)
	if !math.Signbit(zero) {
		t.Fatal("test input lost its sign")
	}
	if got := poseNumber(zero); got != "0" {
		t.Fatalf("negative zero serialized as %q", got)
	}
	p := identityPose()
	p.BDY, p.Tilt, p.EDY = zero, zero, zero
	if got := poseVars(p); len(got) != 0 {
		t.Fatalf("negative-zero identity emitted vars: %v", got)
	}
	l := composeLayout(Traits("negative-zero"))
	_, wrap := bakePose(l, p)
	if wrap != "" {
		t.Fatalf("negative-zero identity emitted wrap: %q", wrap)
	}
	if poseTransforms(l, p, 1) != poseTransforms(l, identityPose(), 1) {
		t.Fatal("negative zero changed identity transforms")
	}
}
