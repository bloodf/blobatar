package blobatar_test

import (
	"testing"

	"github.com/bloodf/blobatar/internal/golden"
)

// Skip-gated parity suites. Each skip names the follow-up epic it waits on.

func TestTraitsPaletteParity(t *testing.T) {
	t.Skip("blobatar epic G2: traits+palette (bloodf/beans#379)")
	_, err := golden.LoadGen2()
	if err != nil {
		t.Fatal(err)
	}
	// TODO(G2): assert Traits/tone/hue for every corpus seed.
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
