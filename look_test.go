package blobatar_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/bloodf/blobatar"
	"github.com/bloodf/blobatar/internal/golden"
)

const lookTestID = "6f1c2a90-bot"

func lookRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func mustJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("json: %v", err)
	}
	return v
}

func TestLookLooksJSON(t *testing.T) {
	root := lookRepoRoot(t)
	rootBytes, err := os.ReadFile(filepath.Join(root, "looks.json"))
	if err != nil {
		t.Fatal(err)
	}
	goldenBytes, err := os.ReadFile(filepath.Join(golden.Root(), "looks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rootBytes, goldenBytes) {
		t.Fatalf("package-root looks.json differs from testdata/golden/looks.json")
	}

	var doc struct {
		States      []string `json:"states"`
		Shapes      []string `json:"shapes"`
		Expressions []string `json:"expressions"`
		Backgrounds []string `json:"backgrounds"`
		Tones       []string `json:"tones"`
	}
	if err := json.Unmarshal(rootBytes, &doc); err != nil {
		t.Fatal(err)
	}

	// Spot-check against appearance.test.ts / appearance.ts const arrays.
	wantStates := []string{"idle", "thinking", "responding", "working", "waiting", "retry", "error"}
	wantShapes := []string{"round", "organic", "boxy", "capsule", "nub", "cloud", "droplet", "hexagon", "sun", "triangle"}
	wantExpr := []string{"idle", "happy", "sad", "mad", "surprised", "wink", "sleepy", "smug", "unsure", "scared", "love", "shy", "sick", "thinking"}
	wantBg := []string{"none", "square", "circle", "squircle"}
	wantTones := []string{"pastel", "pale", "mid", "deep", "bright", "ink"}
	if !reflect.DeepEqual(doc.States, wantStates) {
		t.Fatalf("states: got %#v", doc.States)
	}
	if !reflect.DeepEqual(doc.Shapes, wantShapes) {
		t.Fatalf("shapes: got %#v", doc.Shapes)
	}
	if !reflect.DeepEqual(doc.Expressions, wantExpr) {
		t.Fatalf("expressions: got %#v", doc.Expressions)
	}
	if !reflect.DeepEqual(doc.Backgrounds, wantBg) {
		t.Fatalf("backgrounds: got %#v", doc.Backgrounds)
	}
	if !reflect.DeepEqual(doc.Tones, wantTones) {
		t.Fatalf("tones: got %#v", doc.Tones)
	}
}

func TestLookValidateAcceptsCanonical(t *testing.T) {
	input := mustJSON(t, `{
		"version": 1,
		"base": {
			"shape": "sun",
			"expression": "happy",
			"background": "squircle",
			"hue": 359.5,
			"tone": "ink",
			"palette": {"head": "#AABBCC", "eye": "#000000", "bg": "#FFFFFF"},
			"motion": false
		},
		"states": {
			"error": {"expression": "mad", "palette": {"head": "#FF0000"}},
			"thinking": {}
		}
	}`)
	look, err := blobatar.ValidateLook(input)
	if err != nil {
		t.Fatalf("ValidateLook: %v", err)
	}
	if look.Version != 1 || look.Base.Shape != "sun" || look.Base.Expression != "happy" {
		t.Fatalf("unexpected look: %+v", look)
	}
	if look.Base.Hue == nil || *look.Base.Hue != 359.5 {
		t.Fatalf("hue: %v", look.Base.Hue)
	}
	if look.Base.Motion == nil || *look.Base.Motion {
		t.Fatalf("motion: %v", look.Base.Motion)
	}
	errSt := look.States["error"]
	if errSt.Expression != "mad" || errSt.Palette == nil || errSt.Palette.Head != "#FF0000" {
		t.Fatalf("error state: %+v", errSt)
	}
	if _, ok := look.States["thinking"]; !ok {
		t.Fatal("missing thinking state")
	}
	// Canonical copy — mutating input map must not change the returned Look.
	m := input.(map[string]any)
	m["version"] = float64(2)
	if look.Version != 1 {
		t.Fatal("ValidateLook returned a live alias of the input")
	}
}

func TestLookValidateRejects(t *testing.T) {
	// Ported from packages/beans-blobatar/test/appearance.test.ts reject table.
	cases := []struct {
		name  string
		input any
	}{
		{"non-object", float64(3)},
		{"array", []any{}},
		{"wrong version", mustJSON(t, `{"version":2,"base":{}}`)},
		{"missing base", mustJSON(t, `{"version":1}`)},
		{"unknown top field", mustJSON(t, `{"version":1,"base":{},"extra":1}`)},
		{"unknown appearance field", mustJSON(t, `{"version":1,"base":{"traits":{}}}`)},
		{"unknown state", mustJSON(t, `{"version":1,"base":{},"states":{"sleeping":{}}}`)},
		{"unknown shape", mustJSON(t, `{"version":1,"base":{"shape":"star"}}`)},
		{"unknown expression", mustJSON(t, `{"version":1,"base":{"expression":"angry"}}`)},
		{"unknown background", mustJSON(t, `{"version":1,"base":{"background":"rounded"}}`)},
		{"unknown tone", mustJSON(t, `{"version":1,"base":{"tone":"dark"}}`)},
		{"hue 360", mustJSON(t, `{"version":1,"base":{"hue":360}}`)},
		{"negative hue", mustJSON(t, `{"version":1,"base":{"hue":-1}}`)},
		{"NaN hue", map[string]any{"version": 1, "base": map[string]any{"hue": math.NaN()}}},
		{"infinite hue", map[string]any{"version": 1, "base": map[string]any{"hue": math.Inf(1)}}},
		{"string hue", mustJSON(t, `{"version":1,"base":{"hue":"10"}}`)},
		{"lowercase hex", mustJSON(t, `{"version":1,"base":{"palette":{"head":"#aabbcc"}}}`)},
		{"short hex", mustJSON(t, `{"version":1,"base":{"palette":{"eye":"#ABC"}}}`)},
		{"css color", mustJSON(t, `{"version":1,"base":{"palette":{"bg":"red"}}}`)},
		{"unknown palette key", mustJSON(t, `{"version":1,"base":{"palette":{"cheek":"#FF0000"}}}`)},
		{"null field", mustJSON(t, `{"version":1,"base":{"shape":null}}`)},
		{"null palette", mustJSON(t, `{"version":1,"base":{"palette":null}}`)},
		{"non-boolean motion", mustJSON(t, `{"version":1,"base":{"motion":1}}`)},
		{"null states", mustJSON(t, `{"version":1,"base":{},"states":null}`)},
		{"function", map[string]any{"version": 1, "base": map[string]any{"shape": func() string { return "round" }}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := blobatar.ValidateLook(tc.input)
			if err == nil {
				t.Fatalf("expected rejection for %s", tc.name)
			}
		})
	}
}

func TestLookResolveMissingDefaults(t *testing.T) {
	a, err := blobatar.ResolveAppearance(lookTestID, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if a.Background != "none" || a.Expression != "idle" || !a.Motion {
		t.Fatalf("defaults: %+v", a)
	}
	if a.Shape == "" || a.Tone == "" {
		t.Fatalf("seeded shape/tone empty: %+v", a)
	}
	if !matchHex(a.Palette.Head) || !matchHex(a.Palette.Eye) || !matchHex(a.Palette.Bg) {
		t.Fatalf("palette: %+v", a.Palette)
	}

	b, err := blobatar.ResolveAppearance(lookTestID, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	c, err := blobatar.ResolveAppearance(lookTestID, mustJSON(t, `{"version":1,"base":{}}`), "")
	if err != nil {
		t.Fatal(err)
	}
	if !resolvedEqual(a, b) || !resolvedEqual(a, c) {
		t.Fatalf("nil/null/empty must match:\n%+v\n%+v\n%+v", a, b, c)
	}
}

func TestLookResolveShapesAndTones(t *testing.T) {
	for _, shape := range []string{"round", "organic", "boxy", "capsule", "nub", "cloud", "droplet", "hexagon", "sun", "triangle"} {
		look := mustJSON(t, `{"version":1,"base":{"shape":"`+shape+`"}}`)
		r, err := blobatar.ResolveAppearance(lookTestID, look, "")
		if err != nil {
			t.Fatal(err)
		}
		if r.Shape != shape {
			t.Fatalf("shape %s: got %s", shape, r.Shape)
		}
	}
	for _, tone := range []string{"pastel", "pale", "mid", "deep", "bright", "ink"} {
		look := mustJSON(t, `{"version":1,"base":{"tone":"`+tone+`"}}`)
		r, err := blobatar.ResolveAppearance(lookTestID, look, "")
		if err != nil {
			t.Fatal(err)
		}
		if r.Tone != tone {
			t.Fatalf("tone %s: got %s", tone, r.Tone)
		}
	}
}

func TestLookResolveStatePaletteMerge(t *testing.T) {
	look := mustJSON(t, `{
		"version": 1,
		"base": {
			"shape": "boxy",
			"hue": 120,
			"palette": {"head": "#112233", "eye": "#FFFFFF"},
			"motion": false
		},
		"states": {
			"error": {"expression": "mad", "palette": {"eye": "#EEEEEE"}}
		}
	}`)
	errState, err := blobatar.ResolveAppearance(lookTestID, look, "error")
	if err != nil {
		t.Fatal(err)
	}
	if errState.Shape != "boxy" || errState.Hue != 120 || errState.Expression != "mad" || errState.Motion {
		t.Fatalf("error state: %+v", errState)
	}
	if errState.Palette.Head != "#112233" || errState.Palette.Eye != "#EEEEEE" {
		t.Fatalf("error palette merge: %+v", errState.Palette)
	}

	idle, err := blobatar.ResolveAppearance(lookTestID, look, "idle")
	if err != nil {
		t.Fatal(err)
	}
	if idle.Expression != "idle" || idle.Palette.Eye != "#FFFFFF" {
		t.Fatalf("idle: %+v", idle)
	}

	for _, s := range []string{"idle", "thinking", "responding", "working", "waiting", "retry", "error"} {
		if _, err := blobatar.ResolveAppearance(lookTestID, look, s); err != nil {
			t.Fatalf("state %s: %v", s, err)
		}
	}
}

func TestLookResolveInvalid(t *testing.T) {
	_, err := blobatar.ResolveAppearance(lookTestID, mustJSON(t, `{"version":1,"base":{"hue":400}}`), "")
	if err == nil {
		t.Fatal("expected invalid look error")
	}
	_, err = blobatar.ResolveAppearance(lookTestID, nil, "sleeping")
	if err == nil {
		t.Fatal("expected unknown state error")
	}
}

func TestLookResolveUntouchedIdentity(t *testing.T) {
	a, err := blobatar.ResolveAppearance(lookTestID, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := blobatar.ResolveAppearance(lookTestID, mustJSON(t, `{
		"version": 1,
		"base": {"expression": "happy", "background": "circle"}
	}`), "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Shape != a.Shape || b.Hue != a.Hue {
		t.Fatalf("identity drifted: a=%+v b=%+v", a, b)
	}
	if b.Palette != a.Palette {
		t.Fatalf("palette drifted: a=%+v b=%+v", a.Palette, b.Palette)
	}
	if b.Expression != "happy" || b.Background != "circle" {
		t.Fatalf("overrides missing: %+v", b)
	}
}

func TestLookResolveAppearanceFixtures(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(golden.Root(), "appearance_resolve.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		n++
		var row struct {
			Name     string          `json:"name"`
			BotID    string          `json:"botID"`
			Look     json.RawMessage `json:"look"`
			State    *string         `json:"state"`
			Resolved json.RawMessage `json:"resolved"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("row %d: %v", n, err)
		}
		var look any
		if string(row.Look) != "null" {
			if err := json.Unmarshal(row.Look, &look); err != nil {
				t.Fatalf("%s: look: %v", row.Name, err)
			}
		}
		state := ""
		if row.State != nil {
			state = *row.State
		}
		got, err := blobatar.ResolveAppearance(row.BotID, look, state)
		if err != nil {
			t.Fatalf("%s: %v", row.Name, err)
		}
		var wantObj struct {
			Shape      string  `json:"shape"`
			Expression string  `json:"expression"`
			Background string  `json:"background"`
			Hue        float64 `json:"hue"`
			Tone       string  `json:"tone"`
			Palette    struct {
				Head string `json:"head"`
				Eye  string `json:"eye"`
				Bg   string `json:"bg"`
			} `json:"palette"`
			Motion bool `json:"motion"`
		}
		if err := json.Unmarshal(row.Resolved, &wantObj); err != nil {
			t.Fatalf("%s: want obj: %v", row.Name, err)
		}
		want := blobatar.ResolvedAppearance{
			Shape:      wantObj.Shape,
			Expression: wantObj.Expression,
			Background: wantObj.Background,
			Hue:        wantObj.Hue,
			Tone:       wantObj.Tone,
			Palette:    blobatar.BotPalette{Head: wantObj.Palette.Head, Eye: wantObj.Palette.Eye, Bg: wantObj.Palette.Bg},
			Motion:     wantObj.Motion,
		}
		if !resolvedClose(got, want) {
			t.Fatalf("%s:\n got %+v\nwant %+v", row.Name, got, want)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("no appearance_resolve.jsonl rows")
	}
}

func matchHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for i := 1; i < 7; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func resolvedEqual(a, b blobatar.ResolvedAppearance) bool {
	return a.Shape == b.Shape && a.Expression == b.Expression && a.Background == b.Background &&
		a.Hue == b.Hue && a.Tone == b.Tone && a.Motion == b.Motion && a.Palette == b.Palette
}

func resolvedClose(got, want blobatar.ResolvedAppearance) bool {
	if got.Shape != want.Shape || got.Expression != want.Expression || got.Background != want.Background ||
		got.Tone != want.Tone || got.Motion != want.Motion || got.Palette != want.Palette {
		return false
	}
	return math.Abs(got.Hue-want.Hue) <= 1e-9
}
