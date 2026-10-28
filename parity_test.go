package blobatar_test

import (
	"testing"

	"github.com/bloodf/blobatar/internal/golden"
)

// Skip-gated parity suites. Each skip names the follow-up epic it waits on.

func TestTraitsPaletteParity(t *testing.T) {
	t.Skip("blobatar epic: traits+palette")
	_, err := golden.LoadGen2()
	if err != nil {
		t.Fatal(err)
	}
	// TODO(blobatar epic: traits+palette): assert Traits/tone/hue for every corpus seed.
}

func TestShapeGeometryParity(t *testing.T) {
	t.Skip("blobatar epic: shape geometry")
	// TODO(blobatar epic: shape geometry): compare Geometry() to geometry.jsonl within 1e-9.
}

func TestExpressionsLookParity(t *testing.T) {
	t.Skip("blobatar epic: expressions+look")
	// TODO(blobatar epic: expressions+look): ValidateLook / ResolveAppearance / 14 expressions.
}

func TestSVGRenderParity(t *testing.T) {
	t.Skip("blobatar epic: SVG render")
	g, err := golden.LoadGen2()
	if err != nil {
		t.Fatal(err)
	}
	_ = g
	// TODO(blobatar epic: SVG render): SVG() byte-exact against every gen2.txt markup line.
}

func TestMorphFrameParity(t *testing.T) {
	t.Skip("blobatar epic: morph+frame")
	// TODO(blobatar epic: morph+frame): Lerp and Frame match frames.jsonl within 1e-9.
}
