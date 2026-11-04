package blobatar_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/bloodf/blobatar"
	"github.com/bloodf/blobatar/internal/golden"
)

// This exercises the public production function and compares every output
// field, not a projection or a test-only reconstruction of the assembly.
func compareAllGeometryFixtures(t *testing.T) {
	t.Helper()
	f, err := os.Open(filepath.Join(golden.Root(), "geometry.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	count := 0
	for scanner.Scan() {
		var c struct {
			Seed     string          `json:"seed"`
			Look     any             `json:"look"`
			State    string          `json:"state"`
			Geometry json.RawMessage `json:"geometry"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		t.Run(fmt.Sprintf("row-%d", count), func(t *testing.T) {
			got, err := blobatar.Geometry(c.Seed, c.Look, c.State)
			if err != nil {
				t.Fatal(err)
			}
			assertGeometryJSON(t, got, c.Geometry)
			assertGeometryCounts(t, got)
		})
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 1049 {
		t.Fatalf("geometry oracle: got %d rows, want 1049", count)
	}
}

func assertGeometryJSON(t *testing.T, got blobatar.AvatarGeometry, expected json.RawMessage) {
	t.Helper()
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(expected, &b); err != nil {
		t.Fatal(err)
	}
	var compare func(string, any, any)
	compare = func(path string, got, want any) {
		t.Helper()
		switch w := want.(type) {
		case float64:
			v, ok := got.(float64)
			if !ok || !closeEnough(v, w, floatEps) {
				t.Fatalf("%s: got %v, want %.17g", path, got, w)
			}
		case string, bool, nil:
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s: got %v, want %v", path, got, want)
			}
		case []any:
			v, ok := got.([]any)
			if !ok || len(v) != len(w) {
				t.Fatalf("%s: array mismatch", path)
			}
			for i := range w {
				compare(fmt.Sprintf("%s[%d]", path, i), v[i], w[i])
			}
		case map[string]any:
			v, ok := got.(map[string]any)
			if !ok || len(v) != len(w) {
				t.Fatalf("%s: key mismatch, got %v, want %v", path, got, want)
			}
			for key, value := range w {
				gotValue, exists := v[key]
				if !exists {
					t.Fatalf("%s: missing key %s", path, key)
				}
				compare(path+"."+key, gotValue, value)
			}
		default:
			t.Fatalf("%s: unsupported fixture type %T", path, want)
		}
	}
	compare("geometry", a, b)
}

func assertGeometryCounts(t *testing.T, g blobatar.AvatarGeometry) {
	t.Helper()
	if g.Version != 1 || len(g.Core.Segments) != blobatar.CoreSegments || len(g.Extra.Segments) != blobatar.ExtraSegments ||
		len(g.Background.Path.Segments) != blobatar.BackgroundSegments || len(g.Petals) != blobatar.PetalSlots || len(g.Eyes) != 2 {
		t.Fatal("invalid geometry shape or version")
	}
}

func TestGeometryValidationAndNilLook(t *testing.T) {
	base, err := blobatar.Geometry("seed", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	var nilLook *blobatar.Look
	for _, look := range []any{nilLook, blobatar.Look{Version: 1}, &blobatar.Look{Version: 1}, map[string]any{"version": 1, "base": map[string]any{}}} {
		got, err := blobatar.Geometry("seed", look, "")
		if err != nil || !reflect.DeepEqual(got, base) {
			t.Fatalf("nil/empty look mismatch for %T: %v", look, err)
		}
	}
	badHue := math.NaN()
	for _, c := range []struct {
		look  any
		state string
	}{
		{nil, "bogus"}, {map[string]any{"version": 2, "base": map[string]any{}}, ""},
		{blobatar.Look{Version: 1, Base: blobatar.Appearance{Shape: "bogus"}}, ""},
		{blobatar.Look{Version: 1, Base: blobatar.Appearance{Expression: "bogus"}}, ""},
		{blobatar.Look{Version: 1, Base: blobatar.Appearance{Background: "bogus"}}, ""},
		{blobatar.Look{Version: 1, Base: blobatar.Appearance{Hue: &badHue}}, ""},
		{blobatar.Look{Version: 1, Base: blobatar.Appearance{Palette: &blobatar.BotPalette{Head: "bad"}}}, ""},
	} {
		if got, err := blobatar.Geometry("seed", c.look, c.state); err == nil {
			t.Fatalf("accepted invalid look/state: %+v", c)
		} else if !reflect.DeepEqual(got, blobatar.AvatarGeometry{}) {
			t.Fatal("returned partial geometry on error")
		}
	}
}

func TestGeometryOwnsEndpointsAndRetainsUnposedEyes(t *testing.T) {
	base, err := blobatar.Geometry("identity", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	look := blobatar.Look{Version: 1, Base: blobatar.Appearance{Expression: "thinking"}}
	posed, err := blobatar.Geometry("identity", look, "")
	if err != nil {
		t.Fatal(err)
	}
	if posed.Eyes != base.Eyes || !reflect.DeepEqual(posed.Core, base.Core) {
		t.Fatal("Geometry baked the expression into eyes or silhouette")
	}
	if posed.Pose == base.Pose {
		t.Fatal("expression pose missing")
	}
	mutated, err := blobatar.Geometry("identity", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	mutated.Core.Segments[0][0] = -999
	mutated.Extra.Segments[0][0] = -999
	mutated.Background.Path.Segments[0][0] = -999
	mutated.Petals[0].R = -999
	mutated.Eyes[0].RX = -999
	mutated.Pose.ESX = -999
	again, err := blobatar.Geometry("identity", nil, "")
	if err != nil || !reflect.DeepEqual(again, base) {
		t.Fatal("returned geometry shares mutable endpoint state")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := blobatar.Geometry("identity", nil, "")
			if err != nil || !reflect.DeepEqual(got, base) {
				t.Error("concurrent geometry differed")
			}
		}()
	}
	wg.Wait()
}

func TestGeometryMotionFalseRetainsSeeds(t *testing.T) {
	base, err := blobatar.Geometry("motion", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	motion := false
	look := blobatar.Look{Version: 1, Base: blobatar.Appearance{Motion: &motion}}
	got, err := blobatar.Geometry("motion", look, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Motion {
		t.Fatal("motion:false ignored")
	}
	got.Motion = true
	if !reflect.DeepEqual(got, base) {
		t.Fatal("motion:false changed endpoint data or seeds")
	}
}

func TestGeometryExpressionStateOracle(t *testing.T) {
	f, err := os.Open("testdata/expressions/states.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		var c struct {
			Seed    string                 `json:"seed"`
			Look    blobatar.Look          `json:"look"`
			State   string                 `json:"state"`
			Pose    blobatar.Pose          `json:"pose"`
			Palette blobatar.AvatarPalette `json:"palette"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		t.Run(c.State+"/"+c.Look.States[c.State].Expression, func(t *testing.T) {
			got, err := blobatar.Geometry(c.Seed, c.Look, c.State)
			if err != nil {
				t.Fatal(err)
			}
			if got.Pose != c.Pose || got.Palette != c.Palette {
				t.Fatalf("state expression or palette differs: %+v", got)
			}
			if got.Background.Fill != c.Palette.BG {
				t.Fatal("expression changed background color")
			}
		})
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 98 {
		t.Fatalf("state expression oracle: got %d rows, want 98", count)
	}
}

func TestGeometryStateMergeNormalizationAndJSON(t *testing.T) {
	baseHue, stateHue := 25.0, 359.5
	off, on := false, true
	look := blobatar.Look{
		Version: 1,
		Base: blobatar.Appearance{
			Shape: "sun", Expression: "happy", Background: "circle", Hue: &baseHue, Tone: "pale", Motion: &off,
			Palette: &blobatar.BotPalette{Head: "#AABBCC", Eye: "#000000", Bg: "#FFFFFF"},
		},
		States: map[string]blobatar.Appearance{"error": {
			Shape: "triangle", Expression: "mad", Background: "square", Hue: &stateHue, Tone: "bright", Motion: &on,
			Palette: &blobatar.BotPalette{Eye: "#EEEEEE"},
		}},
	}
	before, err := json.Marshal(look)
	if err != nil {
		t.Fatal(err)
	}
	got, err := blobatar.Geometry(" CAFÉ ", &look, "error")
	if err != nil {
		t.Fatal(err)
	}
	direct := blobatar.Look{Version: 1, Base: blobatar.Appearance{
		Shape: "triangle", Expression: "mad", Background: "square", Hue: &stateHue, Tone: "bright", Motion: &on,
		Palette: &blobatar.BotPalette{Head: "#AABBCC", Eye: "#EEEEEE", Bg: "#FFFFFF"},
	}}
	want, err := blobatar.Geometry("cafe\u0301", direct, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("state merge or normalized seed differs from direct appearance")
	}
	after, err := json.Marshal(look)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("Geometry changed caller's Look")
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var decoded blobatar.AvatarGeometry
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, decoded) {
		t.Fatal("public geometry JSON roundtrip lost fields")
	}
}

func TestGeometryBackgroundModes(t *testing.T) {
	none, err := blobatar.Geometry("background", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if none.Background.Opacity != 0 {
		t.Fatal("default background is not transparent")
	}
	for _, mode := range []string{"none", "circle", "square", "squircle"} {
		got, err := blobatar.Geometry("background", blobatar.Look{Version: 1, Base: blobatar.Appearance{Background: mode}}, "")
		if err != nil {
			t.Fatal(err)
		}
		assertGeometryCounts(t, got)
		if mode == "none" {
			if !reflect.DeepEqual(got, none) {
				t.Fatal("explicit none differs from default")
			}
		} else if got.Background.Opacity != 1 {
			t.Fatalf("%s background is not opaque", mode)
		}
		if mode == "circle" && !reflect.DeepEqual(got.Background.Path, none.Background.Path) {
			t.Fatal("hidden background must retain the circle path")
		}
		if got.Background.Fill != got.Palette.BG {
			t.Fatal("background fill and palette disagree")
		}
		got.Background = none.Background
		if !reflect.DeepEqual(got, none) {
			t.Fatalf("%s background changed the figure", mode)
		}
	}
}
