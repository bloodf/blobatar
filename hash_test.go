package blobatar_test

import (
	"math"
	"sort"
	"testing"

	"github.com/bloodf/blobatar"
	"github.com/bloodf/blobatar/internal/golden"
)

func TestNormalizeNFC(t *testing.T) {
	composed := "caf\u00e9"    // U+00E9
	decomposed := "cafe\u0301" // e + combining acute
	if blobatar.Normalize(composed) != blobatar.Normalize(decomposed) {
		t.Fatalf("NFC pair must normalize equal: %q vs %q",
			blobatar.Normalize(composed), blobatar.Normalize(decomposed))
	}
	if blobatar.Normalize("  Alain@X.com  ") != "alain@x.com" {
		t.Fatalf("trim+lower: got %q", blobatar.Normalize("  Alain@X.com  "))
	}
}

func TestSeedStateNFCPair(t *testing.T) {
	composed := "caf\u00e9"
	decomposed := "cafe\u0301"
	a := blobatar.SeedStateForTest(composed)
	b := blobatar.SeedStateForTest(decomposed)
	if a != b {
		t.Fatalf("NFC pair must hash equal: %d vs %d", a, b)
	}
}

func TestSeedStateKnown(t *testing.T) {
	// Values observed from Beans vendor/core/hash.ts under bun (>>> 0).
	cases := []struct {
		seed string
		want uint32
	}{
		{"", 1779033703},
		{"alain", 2106776898},
		{"alaim", 3085476782},
		{"🦊", 1726804061},
		{"caf\u00e9", 3438182198},
		{"cafe\u0301", 3438182198},
		{"histogram-0", 2784293965},
	}
	for _, tc := range cases {
		if got := blobatar.SeedStateForTest(tc.seed); got != tc.want {
			t.Errorf("seedState(%q)=%d want %d", tc.seed, got, tc.want)
		}
	}
}

func TestStreamShapeAlain(t *testing.T) {
	// bun: stream(seedState("alain"), "shape") === 0.1506984510924667
	got := blobatar.StreamForTest(blobatar.SeedStateForTest("alain"), "shape")
	const want = 0.1506984510924667
	if got != want {
		t.Fatalf("stream(alain, shape)=%v want %v (Δ=%g)", got, want, math.Abs(got-want))
	}
}

func TestHistogramMatchesGen2(t *testing.T) {
	g, err := golden.LoadGen2()
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	const n = 20_000
	for i := 0; i < n; i++ {
		seed := "histogram-" + itoa(i)
		counts[blobatar.ShapeForSeed(seed)]++
	}
	var names []string
	for name := range g.Histogram {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		want := g.Histogram[name]
		got := counts[name]
		if got != want {
			t.Errorf("shape %s: got %d want %d", name, got, want)
		}
	}
	if len(counts) != len(g.Histogram) {
		t.Errorf("shape set mismatch: got %v want %v", keys(counts), names)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

func keys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
