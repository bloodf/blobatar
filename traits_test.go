package blobatar_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/bloodf/blobatar"
)

func TestTraitsOverrides(t *testing.T) {
	base := blobatar.Traits("alain").Float("shape")
	fixed := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": blobatar.FixedTrait(0.965)}).Float("shape")
	if fixed != 0.965 {
		t.Fatalf("fixed override: got %v want 0.965", fixed)
	}
	if base == fixed {
		t.Fatalf("override should change shape float")
	}

	// Clamp: 1 → 0.999999, negative → 0, NaN → 0.
	if got := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": blobatar.FixedTrait(1.0)}).Float("shape"); got != 0.999999 {
		t.Fatalf("clamp 1: got %v", got)
	}
	if got := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": blobatar.FixedTrait(-0.5)}).Float("shape"); got != 0 {
		t.Fatalf("clamp neg: got %v", got)
	}
	if got := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": blobatar.FixedTrait(math.NaN())}).Float("shape"); got != 0 {
		t.Fatalf("clamp NaN: got %v", got)
	}

	// Empty list falls through to the hash.
	empty := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": blobatar.ListTrait()}).Float("shape")
	if empty != base {
		t.Fatalf("empty list: got %v want %v", empty, base)
	}

	list := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": blobatar.ListTrait(0.11, 0.965)}).Float("shape")
	if list != 0.11 && list != 0.965 {
		t.Fatalf("list pick: got %v", list)
	}
}

func TestTraitOverrideJSON(t *testing.T) {
	var fixed blobatar.TraitOverride
	if err := json.Unmarshal([]byte(`0.965`), &fixed); err != nil {
		t.Fatal(err)
	}
	if got := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": fixed}).Float("shape"); got != 0.965 {
		t.Fatalf("JSON fixed: got %v", got)
	}

	var list blobatar.TraitOverride
	if err := json.Unmarshal([]byte(`[0.11, 0.965]`), &list); err != nil {
		t.Fatal(err)
	}
	got := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": list}).Float("shape")
	if got != 0.11 && got != 0.965 {
		t.Fatalf("JSON list: got %v", got)
	}

	// Wrong JSON types must be rejected (not silently ignored).
	for _, raw := range []string{`"nope"`, `true`, `null`, `{}`, `{"x":1}`, `[true]`, `["0.5"]`} {
		var o blobatar.TraitOverride
		if err := json.Unmarshal([]byte(raw), &o); err == nil {
			t.Errorf("Unmarshal(%s) = %+v, want error", raw, o)
		}
	}

	// Round-trip matches Beans number | number[] shape.
	for _, o := range []blobatar.TraitOverride{
		blobatar.FixedTrait(0.5),
		blobatar.ListTrait(0.1, 0.2),
		blobatar.ListTrait(),
	} {
		b, err := json.Marshal(o)
		if err != nil {
			t.Fatal(err)
		}
		var back blobatar.TraitOverride
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("round-trip %s: %v", b, err)
		}
		b2, err := json.Marshal(back)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != string(b2) {
			t.Fatalf("round-trip mismatch: %s vs %s", b, b2)
		}
	}

	var m blobatar.TraitOverrides
	if err := json.Unmarshal([]byte(`{"shape":0.965,"eye.gap":[0.1,0.2]}`), &m); err != nil {
		t.Fatal(err)
	}
	if got := blobatar.NewTraits("alain", true, m).Float("shape"); got != 0.965 {
		t.Fatalf("map JSON shape: got %v", got)
	}
}

func TestTraitsNumIntBoolJitter(t *testing.T) {
	tr := blobatar.Traits("alain")
	hue := tr.Num("hue", 0, 360)
	if hue < 0 || hue >= 360 {
		t.Fatalf("hue out of range: %v", hue)
	}
	if math.Abs(hue-113.5819791443646) > 1e-9 {
		t.Fatalf("hue: got %v", hue)
	}

	n := tr.Int("shape", 0, 9)
	if n < 0 || n > 9 {
		t.Fatalf("int out of range: %d", n)
	}

	_ = tr.Bool("motion")
	_ = tr.Bool("motion", 0.25)
	j := tr.Jitter("eye.gap", 0.1)
	if j < -0.1 || j >= 0.1 {
		t.Fatalf("jitter out of range: %v", j)
	}
}

func TestIntMaxLessThanMin(t *testing.T) {
	// Matches Beans traits.ts: min + floor(Float(key) * (max - min + 1)).
	// Not an error when max < min; result may lie outside both endpoints.
	cases := []struct {
		seed     string
		key      string
		min, max int
		want     int
	}{
		{"alain", "k", 5, 3, 4},
		{"alain", "k", 0, -1, 0},
		{"alain", "k", 10, 0, 8},
		{"alain", "k", 3, 3, 3},
		{"bob", "k", 5, 3, 4},
		{"bob", "k", 10, 0, 4},
		{"x", "k", 5, 3, 4},
		{"x", "k", 10, 0, 5},
	}
	for _, tc := range cases {
		tr := blobatar.Traits(tc.seed)
		f := tr.Float(tc.key)
		got := tr.Int(tc.key, tc.min, tc.max)
		formula := tc.min + int(math.Floor(f*float64(tc.max-tc.min+1)))
		if got != formula {
			t.Fatalf("%s Int(%q,%d,%d)=%d, formula=%d (f=%v)", tc.seed, tc.key, tc.min, tc.max, got, formula, f)
		}
		if got != tc.want {
			t.Fatalf("%s Int(%q,%d,%d)=%d want %d (Beans bun 1.4.4 @ 4760281f)",
				tc.seed, tc.key, tc.min, tc.max, got, tc.want)
		}
	}
}

func TestToneNameBands(t *testing.T) {
	cases := []struct {
		v    float64
		want string
	}{
		{0, "pastel"},
		{0.199999, "pastel"},
		{0.2, "pale"},
		{0.35, "pale"},
		{0.36, "mid"},
		{0.62, "deep"},
		{0.8, "bright"},
		{0.93, "ink"},
		{0.999, "ink"},
	}
	for _, tc := range cases {
		if got := blobatar.ToneName(tc.v); got != tc.want {
			t.Errorf("ToneName(%v)=%q want %q", tc.v, got, tc.want)
		}
	}
}
