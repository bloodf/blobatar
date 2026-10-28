// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/traits.ts and the shape band table
// in vendor/core/styles/blob.ts), MIT License. See NOTICE for the full license text.

package blobatar

// Shape band upper edges from vendor/core/styles/blob.ts BANDS, in order.
// pick(v) = first band with v < upper.
var shapeBands = []struct {
	name  string
	upper float64
}{
	{"round", 0.22},
	{"organic", 0.48},
	{"boxy", 0.6},
	{"capsule", 0.7},
	{"nub", 0.79},
	{"cloud", 0.86},
	{"droplet", 0.915},
	{"hexagon", 0.95},
	{"sun", 0.98},
	{"triangle", 1},
}

// traitFloat returns the unit float for key under seed (normalized).
func traitFloat(seed, key string) float64 {
	return stream(seedState(seed, true), key)
}

// shapeForSeed returns the silhouette name the gen2 band table assigns to seed.
// Only as much of Traits/layout as the histogram assertion needs.
func shapeForSeed(seed string) string {
	v := traitFloat(seed, "shape")
	for _, b := range shapeBands {
		if v < b.upper {
			return b.name
		}
	}
	return shapeBands[len(shapeBands)-1].name
}
