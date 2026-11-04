package blobatar

import (
	"fmt"
	"math"
	"strconv"
	"testing"
)

// Expected strings follow the pinned JavaScript's two operations: multiply
// in binary64, then Math.round. A decimal-looking half need not be a binary
// half, and even Nextafter can still multiply onto the same rounded half.
type roundingBoundary struct {
	value float64
	below string
	at    string
	above string
}

var shapeRoundingBoundaries = []roundingBoundary{
	// Coordinates from the macOS/arm64 path failures.
	{29.075, "29.07", "29.08", "29.08"},
	{-29.075, "-29.08", "-29.07", "-29.07"},
	{57.425, "57.42", "57.43", "57.43"},
	{-57.425, "-57.42", "-57.42", "-57.42"},
	{77.255, "77.25", "77.26", "77.26"},
	{-77.255, "-77.26", "-77.25", "-77.25"},
	{0.005, "0", "0.01", "0.01"},
	{-0.005, "-0.01", "0", "0"},
	{1.125, "1.12", "1.13", "1.13"},
	{-1.125, "-1.13", "-1.12", "-1.12"},
	{2.005, "2", "2.01", "2.01"},
	{-2.005, "-2.01", "-2", "-2"},
	{2.675, "2.67", "2.68", "2.68"},
	// The next float below -2.675 still multiplies to exactly -267.5.
	{-2.675, "-2.67", "-2.67", "-2.67"},
}

var poseRoundingBoundaries = []roundingBoundary{
	{0.0005, "0", "0.001", "0.001"},
	{-0.0005, "-0.001", "0", "0"},
	{1.0625, "1.062", "1.063", "1.063"},
	{-1.0625, "-1.063", "-1.062", "-1.062"},
	{1.2345, "1.234", "1.235", "1.235"},
	{-1.2345, "-1.235", "-1.234", "-1.234"},
}

func TestShapeNumberBinary64HalfBoundaries(t *testing.T) {
	checkRoundingBoundaries(t, shapeNumber, shapeRoundingBoundaries)
}

func TestPoseNumberBinary64HalfBoundaries(t *testing.T) {
	checkRoundingBoundaries(t, poseNumber, poseRoundingBoundaries)
}

func checkRoundingBoundaries(t *testing.T, round func(float64) string, cases []roundingBoundary) {
	t.Helper()
	for _, c := range cases {
		values := [3]float64{math.Nextafter(c.value, math.Inf(-1)), c.value, math.Nextafter(c.value, math.Inf(1))}
		wants := [3]string{c.below, c.at, c.above}
		for i, v := range values {
			t.Run(fmt.Sprintf("%g/%d", c.value, i), func(t *testing.T) {
				if got := round(v); got != wants[i] {
					t.Errorf("round(%.17g): got %q, want %q", v, got, wants[i])
				}
			})
		}
	}
}

func TestGeometryScaledHalfBoundaries(t *testing.T) {
	for _, group := range []struct {
		scale float64
		cases []roundingBoundary
	}{{100, shapeRoundingBoundaries}, {1000, poseRoundingBoundaries}} {
		for _, c := range group.cases {
			values := [3]float64{math.Nextafter(c.value, math.Inf(-1)), c.value, math.Nextafter(c.value, math.Inf(1))}
			wants := [3]string{c.below, c.at, c.above}
			for i, v := range values {
				want, err := strconv.ParseFloat(wants[i], 64)
				if err != nil {
					t.Fatal(err)
				}
				// Deliberately leave the multiplication at the caller: the helper's
				// conversion must still prevent fusion when it is inlined here.
				if got := roundGeometry(v*group.scale) / group.scale; got != want {
					t.Errorf("scale %g, input %.17g: got %g, want %g", group.scale, v, got, want)
				}
			}
		}
	}
}
