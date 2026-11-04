package blobatar

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"
)

// Generated only from the TS source pinned in testdata/golden/SOURCE. This
// checks raw numeric layout to 1e-9 and serialized core/eye/extra paths exactly.
func TestLayoutMatchesBeans(t *testing.T) {
	seen := map[string]int{}
	count := readLayoutRows(t, "testdata/layout/layout.jsonl", func(raw []byte) {
		var c struct {
			Name      string          `json:"name"`
			Seed      string          `json:"seed"`
			Overrides TraitOverrides  `json:"overrides"`
			Expected  json.RawMessage `json:"expected"`
			CorePath  string          `json:"corePath"`
			EyePaths  [2]string       `json:"eyePaths"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(t *testing.T) {
			got := composeLayout(NewTraits(c.Seed, true, c.Overrides))
			assertLayoutJSON(t, got, c.Expected)
			if got.corePath() != c.CorePath {
				t.Errorf("core path differs:\ngot  %s\nwant %s", got.corePath(), c.CorePath)
			}
			for i, eye := range got.Eyes {
				if eye.path() != c.EyePaths[i] {
					t.Errorf("eye %d path differs:\ngot  %s\nwant %s", i, eye.path(), c.EyePaths[i])
				}
			}
			seen[got.Shape]++
		})
	})
	if count != 1053 {
		t.Fatalf("layout oracle: got %d rows, want 1053", count)
	}
	for _, shape := range shapes {
		if seen[shape] == 0 {
			t.Errorf("missing shape coverage for %s", shape)
		}
	}
}

// Validate the layout-to-contour chain against the original, unchanged
// Geometry fixture. Only its implemented fields are checked; background,
// expression pose, palette, motion and seeds remain part of the G3 work.
func TestLayoutGeometryGoldenSubset(t *testing.T) {
	shapePosition := map[string]float64{
		"round": 0.11, "organic": 0.35, "boxy": 0.54, "capsule": 0.65, "nub": 0.745,
		"cloud": 0.825, "droplet": 0.8875, "hexagon": 0.9325, "sun": 0.965, "triangle": 0.99,
	}
	row := 0
	count := readLayoutRows(t, "testdata/golden/geometry.jsonl", func(raw []byte) {
		var c struct {
			Seed     string `json:"seed"`
			Look     *Look  `json:"look"`
			State    string `json:"state"`
			Geometry struct {
				Core   json.RawMessage `json:"core"`
				Extra  json.RawMessage `json:"extra"`
				Petals json.RawMessage `json:"petals"`
				Eyes   json.RawMessage `json:"eyes"`
			} `json:"geometry"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatal(err)
		}
		overrides := TraitOverrides{}
		if c.Look != nil {
			a := effective(*c.Look, c.State)
			if a.Shape != "" {
				v, ok := shapePosition[a.Shape]
				if !ok {
					t.Fatalf("unknown fixture shape %q", a.Shape)
				}
				overrides["shape"] = FixedTrait(v)
			}
		}
		l := composeLayout(NewTraits(c.Seed, true, overrides))
		centre := point{l.Body.CX, l.Body.CY}
		normalize := func(t *testing.T, path string, count int) cubicPath {
			t.Helper()
			curves, err := pathCubics(path)
			if err != nil {
				t.Fatal(err)
			}
			result, err := normalizeContour(curves, count, centre)
			if err != nil {
				t.Fatal(err)
			}
			return result
		}
		// cubicPath predates the JSON boundary; give the test projection the
		// same lowercase fields as Geometry without changing that internal type.
		pathJSON := func(path cubicPath) any {
			return struct {
				Start    point        `json:"start"`
				Segments [][6]float64 `json:"segments"`
			}{path.Start, path.Segments}
		}
		t.Run(fmt.Sprintf("row-%d/%s", row, l.Shape), func(t *testing.T) {
			assertLayoutJSON(t, pathJSON(normalize(t, l.corePath(), 24)), c.Geometry.Core)
			extra := cubicPath{Start: centre, Segments: make([][6]float64, 4)}
			for i := range extra.Segments {
				extra.Segments[i] = [6]float64{centre[0], centre[1], centre[0], centre[1], centre[0], centre[1]}
			}
			if len(l.Extra) == 1 {
				extra = normalize(t, l.Extra[0], 4)
			}
			if len(l.Extra) > 1 {
				t.Fatal("unexpected extra outline count")
			}
			assertLayoutJSON(t, pathJSON(extra), c.Geometry.Extra)
			petals := make([]layoutCircle, 9)
			for i := range petals {
				petals[i] = layoutCircle{centre[0], centre[1], 0}
			}
			for i, p := range l.Petals {
				if i >= len(petals) {
					t.Fatal("too many petals")
				}
				round := func(v float64) float64 {
					r, err := strconv.ParseFloat(shapeNumber(v), 64)
					if err != nil {
						t.Fatal(err)
					}
					return r
				}
				petals[i] = layoutCircle{round(p.CX), round(p.CY), round(p.R)}
			}
			assertLayoutJSON(t, petals, c.Geometry.Petals)
			assertLayoutJSON(t, l.Eyes, c.Geometry.Eyes)
		})
		row++
	})
	if count != 1049 {
		t.Fatalf("geometry oracle: got %d rows, want 1049", count)
	}
}

func readLayoutRows(t *testing.T, path string, visit func([]byte)) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	count := 0
	for scanner.Scan() {
		visit(scanner.Bytes())
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return count
}

// JSON comparison keeps nested shape-specific fields and every array element
// in scope. Missing/extra fields, non-finite output and strings all fail.
func assertLayoutJSON(t *testing.T, got any, want json.RawMessage) {
	t.Helper()
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &b); err != nil {
		t.Fatal(err)
	}
	var compare func(string, any, any)
	compare = func(path string, a, b any) {
		t.Helper()
		switch want := b.(type) {
		case float64:
			v, ok := a.(float64)
			if !ok || !(math.Abs(v-want) <= 1e-9) {
				t.Fatalf("%s: got %v, want %.17g", path, a, want)
			}
		case string:
			if a != want {
				t.Fatalf("%s: got %v, want %s", path, a, want)
			}
		case []any:
			v, ok := a.([]any)
			if !ok || len(v) != len(want) {
				t.Fatalf("%s: array mismatch: got %v, want %v", path, a, want)
			}
			for i := range want {
				compare(fmt.Sprintf("%s[%d]", path, i), v[i], want[i])
			}
		case map[string]any:
			v, ok := a.(map[string]any)
			if !ok || len(v) != len(want) {
				t.Fatalf("%s: object mismatch: got %v, want %v", path, a, want)
			}
			for k, value := range want {
				compare(path+"."+k, v[k], value)
			}
		default:
			t.Fatalf("%s: unexpected fixture value %T", path, b)
		}
	}
	compare("layout", a, b)
}
