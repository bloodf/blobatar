package blobatar_test

import (
	"math"
	"testing"

	"github.com/bloodf/blobatar"
)

func TestTraitsOverrides(t *testing.T) {
	base := blobatar.Traits("alain").Float("shape")
	fixed := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": 0.965}).Float("shape")
	if fixed != 0.965 {
		t.Fatalf("fixed override: got %v want 0.965", fixed)
	}
	if base == fixed {
		t.Fatalf("override should change shape float")
	}

	// Clamp: 1 → 0.999999, negative → 0, NaN → 0.
	if got := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": 1.0}).Float("shape"); got != 0.999999 {
		t.Fatalf("clamp 1: got %v", got)
	}
	if got := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": -0.5}).Float("shape"); got != 0 {
		t.Fatalf("clamp neg: got %v", got)
	}
	if got := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": math.NaN()}).Float("shape"); got != 0 {
		t.Fatalf("clamp NaN: got %v", got)
	}

	// Empty list falls through to the hash.
	empty := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": []float64{}}).Float("shape")
	if empty != base {
		t.Fatalf("empty list: got %v want %v", empty, base)
	}

	list := blobatar.NewTraits("alain", true, blobatar.TraitOverrides{"shape": []float64{0.11, 0.965}}).Float("shape")
	if list != 0.11 && list != 0.965 {
		t.Fatalf("list pick: got %v", list)
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
