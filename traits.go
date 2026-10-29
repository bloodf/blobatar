// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/traits.ts and the shape band table
// in vendor/core/styles/blob.ts), MIT License. See NOTICE for the full license text.

package blobatar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
)

// TraitOverride is a fixed unit float or a candidate list, matching Beans
// `TraitOverrides = Record<string, number | number[]>`.
//
// Construct with FixedTrait or ListTrait. JSON encodes as a number or a
// number array; values of any other JSON type are rejected.
type TraitOverride struct {
	kind overrideKind
	num  float64
	list []float64
}

type overrideKind byte

const (
	overrideFixed overrideKind = iota
	overrideList
)

// FixedTrait returns a single-value override (Beans number form).
func FixedTrait(v float64) TraitOverride {
	return TraitOverride{kind: overrideFixed, num: v}
}

// ListTrait returns a candidate-list override (Beans number[] form).
// An empty list is the same as omitting the key (falls through to the hash).
func ListTrait(vs ...float64) TraitOverride {
	list := make([]float64, len(vs))
	copy(list, vs)
	return TraitOverride{kind: overrideList, list: list}
}

// UnmarshalJSON accepts a JSON number or array of numbers only.
func (o *TraitOverride) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("blobatar: TraitOverride: empty JSON")
	}
	if string(data) == "null" {
		return fmt.Errorf("blobatar: TraitOverride: want number or number[], got null")
	}
	if data[0] == '[' {
		var list []json.RawMessage
		if err := json.Unmarshal(data, &list); err != nil {
			return fmt.Errorf("blobatar: TraitOverride: %w", err)
		}
		nums := make([]float64, len(list))
		for i, raw := range list {
			raw = bytes.TrimSpace(raw)
			// Reject JSON null elements (encoding/json would decode null as 0).
			if string(raw) == "null" {
				return fmt.Errorf("blobatar: TraitOverride: array element %d: want number, got null", i)
			}
			var n float64
			if err := json.Unmarshal(raw, &n); err != nil {
				return fmt.Errorf("blobatar: TraitOverride: array element %d: want number, got %s", i, raw)
			}
			nums[i] = n
		}
		*o = ListTrait(nums...)
		return nil
	}
	var num float64
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("blobatar: TraitOverride: want number or number[], got %s", data)
	}
	*o = FixedTrait(num)
	return nil
}

// MarshalJSON emits a JSON number or number array to match Beans.
func (o TraitOverride) MarshalJSON() ([]byte, error) {
	switch o.kind {
	case overrideFixed:
		return json.Marshal(o.num)
	case overrideList:
		if o.list == nil {
			return []byte("[]"), nil
		}
		return json.Marshal(o.list)
	default:
		return nil, fmt.Errorf("blobatar: TraitOverride: invalid kind %d", o.kind)
	}
}

// TraitOverrides maps trait keys to a fixed unit float in [0, 1) or a list of
// candidate unit floats. A list means "any of these"; the key's own hash picks
// the index. An empty list is the same as omitting the key.
type TraitOverrides map[string]TraitOverride

// TraitReader addresses every value by a string key rather than drawing from a
// sequential stream, so trait keys are an append-only namespace.
type TraitReader struct {
	state     uint32
	overrides TraitOverrides
}

// Traits returns a trait reader for seed with Normalize applied and no overrides.
func Traits(seed string) *TraitReader {
	return NewTraits(seed, true, nil)
}

// NewTraits returns a trait reader. When normalize is true, the seed is passed
// through Normalize first (matching vendor/core/traits.ts).
func NewTraits(seed string, normalize bool, overrides TraitOverrides) *TraitReader {
	return &TraitReader{state: seedState(seed, normalize), overrides: overrides}
}

// Float returns a uniform float in [0, 1) for key.
func (t *TraitReader) Float(key string) float64 {
	if t.overrides != nil {
		if o, ok := t.overrides[key]; ok {
			switch o.kind {
			case overrideFixed:
				return clampUnit(o.num)
			case overrideList:
				if len(o.list) == 0 {
					break
				}
				i := int(math.Floor(stream(t.state, key) * float64(len(o.list))))
				if i >= len(o.list) {
					i = len(o.list) - 1
				}
				return clampUnit(o.list[i])
			default:
				// Exhaustiveness: newly added overrideKind values must be handled.
				panic(fmt.Sprintf("blobatar: unhandled TraitOverride kind %d", o.kind))
			}
		}
	}
	return stream(t.state, key)
}

func clampUnit(o float64) float64 {
	// Matches traits.ts: o > 0 ? (o < 1 ? o : 0.999999) : 0
	// NaN comparisons are false, so NaN becomes 0.
	if o > 0 {
		if o < 1 {
			return o
		}
		return 0.999999
	}
	return 0
}

// Num returns a uniform float in [min, max).
func (t *TraitReader) Num(key string, min, max float64) float64 {
	return min + t.Float(key)*(max-min)
}

// Int returns a uniform integer in [min, max] when max >= min.
//
// When max < min, behavior matches Beans vendor/core/traits.ts exactly:
//
//	min + floor(Float(key) * (max - min + 1))
//
// That expression is not an error; the result can fall outside both [min, max]
// and [max, min]. Callers that need a closed range must ensure max >= min.
func (t *TraitReader) Int(key string, min, max int) int {
	return min + int(math.Floor(t.Float(key)*float64(max-min+1)))
}

// PickIndex returns the option index for key over n options (n > 0).
func (t *TraitReader) PickIndex(key string, n int) int {
	if n <= 0 {
		return 0
	}
	i := int(math.Floor(t.Float(key) * float64(n)))
	if i >= n {
		return n - 1
	}
	return i
}

// Bool reports true with probability p (default 0.5 when p is omitted).
func (t *TraitReader) Bool(key string, p ...float64) bool {
	prob := 0.5
	if len(p) > 0 {
		prob = p[0]
	}
	return t.Float(key) < prob
}

// Jitter returns symmetric jitter in [-amount, amount).
func (t *TraitReader) Jitter(key string, amount float64) float64 {
	return (t.Float(key)*2 - 1) * amount
}

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

// toneEdges are the upper edges of the tone swatches in vendor/core/color.ts.
// Each band excludes its edge. Names mirror appearance.ts BOT_TONES.
var toneEdges = []float64{0.2, 0.36, 0.62, 0.8, 0.93, 1}

// toneNames are the BOT_TONES labels for toneEdges.
var toneNames = []string{"pastel", "pale", "mid", "deep", "bright", "ink"}

// ToneBandEdges returns a copy of the tone swatch upper edges.
func ToneBandEdges() []float64 {
	out := make([]float64, len(toneEdges))
	copy(out, toneEdges)
	return out
}

// ToneBandNames returns a copy of the BOT_TONES labels for ToneBandEdges.
func ToneBandNames() []string {
	out := make([]string, len(toneNames))
	copy(out, toneNames)
	return out
}

// ToneName maps a unit tone float to its swatch name.
func ToneName(v float64) string {
	for i, edge := range toneEdges {
		if v < edge {
			return toneNames[i]
		}
	}
	return toneNames[len(toneNames)-1]
}

// Hue returns the seed's hue in [0, 360).
func Hue(seed string) float64 {
	return Traits(seed).Num("hue", 0, 360)
}

// Tone returns the seed's unit tone float in [0, 1).
func Tone(seed string) float64 {
	return Traits(seed).Float("tone")
}

// SeedPalette returns hue, tone, tone name, and the enforced palette for seed.
func SeedPalette(seed string) (hue, tone float64, toneName string, p Palette, err error) {
	t := Traits(seed)
	hue = t.Num("hue", 0, 360)
	tone = t.Float("tone")
	toneName = ToneName(tone)
	p, err = PaletteColors(hue, true, tone)
	return
}

// shapeForSeed returns the silhouette name the gen2 band table assigns to seed.
func shapeForSeed(seed string) string {
	v := Traits(seed).Float("shape")
	for _, b := range shapeBands {
		if v < b.upper {
			return b.name
		}
	}
	return shapeBands[len(shapeBands)-1].name
}
