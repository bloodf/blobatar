package blobatar

import (
	"math"
	"testing"
)

func TestGeometryRoundMatchesJSTies(t *testing.T) {
	for _, c := range []struct{ input, expected float64 }{
		{-1.5, -1}, {-0.5, math.Copysign(0, -1)}, {math.Nextafter(0.5, 0), 0},
		{0.5, 1}, {1.5, 2}, {-1.5000001, -2}, {1.4999999, 1},
	} {
		got := roundGeometry(c.input)
		if got != c.expected || (got == 0 && math.Signbit(got) != math.Signbit(c.expected)) {
			t.Errorf("round(%g): got %g, want %g", c.input, got, c.expected)
		}
	}
}

func TestGeometryContourErrors(t *testing.T) {
	if _, err := geometryContour("M0 0X1 2", 4, point{}); err == nil {
		t.Fatal("accepted an invalid contour")
	}
	if _, err := geometryContour("M0 0H10V10H0Z", 1, point{}); err == nil {
		t.Fatal("accepted too many segments")
	}
}
