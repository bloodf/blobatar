package blobatar

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

func TestSuperellipseMatchesBeans(t *testing.T) {
	f, e := os.Open("testdata/contour/superellipse.jsonl")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	count := 0
	for sc.Scan() {
		var c struct {
			Input struct {
				CX  float64 `json:"cx"`
				CY  float64 `json:"cy"`
				RX  float64 `json:"rx"`
				RY  float64 `json:"ry"`
				N   float64 `json:"n"`
				Rot float64 `json:"rot"`
			} `json:"input"`
			Expected string `json:"expected"`
		}
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		x := c.Input
		got := superellipse(x.CX, x.CY, x.RX, x.RY, x.N, x.Rot)
		if got != c.Expected {
			t.Errorf("case %+v\ngot  %s\nwant %s", x, got, c.Expected)
		}
		count++
	}
	if e := sc.Err(); e != nil {
		t.Fatal(e)
	}
	if count != 4 {
		t.Fatalf("got %d cases", count)
	}
}

func TestOtherShapesMatchBeans(t *testing.T) {
	f, e := os.Open("testdata/contour/shapes.jsonl")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	count := 0
	for sc.Scan() {
		var c struct {
			Kind     string            `json:"kind"`
			Args     []json.RawMessage `json:"args"`
			Expected string            `json:"expected"`
		}
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		num := func(i int) float64 {
			var x float64
			if err := json.Unmarshal(c.Args[i], &x); err != nil {
				t.Fatal(err)
			}
			return x
		}
		var got string
		switch c.Kind {
		case "blob":
			var radii []float64
			if err := json.Unmarshal(c.Args[4], &radii); err != nil {
				t.Fatal(err)
			}
			got = blobPath(num(0), num(1), num(2), num(3), radii, num(5))
		case "polygon":
			got = polygonPath(num(0), num(1), num(2), num(3), int(num(4)), num(5), num(6))
		case "box":
			got = boxPath(num(0), num(1), num(2), num(3))
		case "taper":
			got = taperPath(num(0), num(1), num(2), num(3), num(4))
		default:
			t.Fatalf("unexpected kind %q", c.Kind)
		}
		if got != c.Expected {
			t.Errorf("%s: got %q, want %q", c.Kind, got, c.Expected)
		}
		count++
	}
	if e := sc.Err(); e != nil {
		t.Fatal(e)
	}
	if count != 6 {
		t.Fatalf("got %d cases", count)
	}
}

func TestShapeNumberRoundNearHalf(t *testing.T) {
	for _, tc := range []struct {
		v    float64
		want string
	}{{0.004999999999999999, "0"}, {0.005, "0.01"}, {-0.005, "0"}, {-0.005000000000000001, "-0.01"}} {
		if got := shapeNumber(tc.v); got != tc.want {
			t.Errorf("shapeNumber(%v) = %s, want %s", tc.v, got, tc.want)
		}
	}
}
