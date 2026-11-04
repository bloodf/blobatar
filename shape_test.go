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
