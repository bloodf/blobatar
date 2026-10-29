// Package golden reads the committed fixture files under testdata/golden.
package golden

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Root returns the absolute path to testdata/golden.
func Root() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("golden: runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "golden"))
}

// Histogram is shape → count from gen2.txt.
type Histogram map[string]int

// Gen2 holds the parsed gen2.txt sections.
type Gen2 struct {
	Histogram Histogram
	Markup    map[string]string
	Hashes    map[string]string
}

// LoadGen2 parses testdata/golden/gen2.txt.
func LoadGen2() (Gen2, error) {
	raw, err := os.ReadFile(filepath.Join(Root(), "gen2.txt"))
	if err != nil {
		return Gen2{}, err
	}
	out := Gen2{
		Histogram: Histogram{},
		Markup:    map[string]string{},
		Hashes:    map[string]string{},
	}
	var section string
	sc := bufio.NewScanner(bytes.NewReader(raw))
	// Markup lines can be long SVGs.
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			return Gen2{}, fmt.Errorf("golden: line has no tab: %q", line)
		}
		k, v := line[:tab], line[tab+1:]
		switch section {
		case "histogram":
			n, err := strconv.Atoi(v)
			if err != nil {
				return Gen2{}, fmt.Errorf("golden: histogram count for %q: %w", k, err)
			}
			out.Histogram[k] = n
		case "markup":
			out.Markup[k] = v
		case "hashes":
			out.Hashes[k] = v
		default:
			return Gen2{}, fmt.Errorf("golden: unknown section %q", section)
		}
	}
	return out, sc.Err()
}

// NormalizeCase is one row of normalize.jsonl.
type NormalizeCase struct {
	Name       string `json:"name"`
	Seed       string `json:"seed"`
	Normalized string `json:"normalized"`
}

// LoadNormalize parses testdata/golden/normalize.jsonl.
func LoadNormalize() ([]NormalizeCase, error) {
	raw, err := os.ReadFile(filepath.Join(Root(), "normalize.jsonl"))
	if err != nil {
		return nil, err
	}
	var out []NormalizeCase
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var c NormalizeCase
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return nil, fmt.Errorf("golden: normalize.jsonl: %w", err)
		}
		out = append(out, c)
	}
	return out, sc.Err()
}

// SourceField reads a key=value field from testdata/golden/SOURCE.
func SourceField(key string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(Root(), "SOURCE"))
	if err != nil {
		return "", err
	}
	prefix := key + "="
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix), nil
		}
	}
	return "", fmt.Errorf("golden: %s missing from SOURCE", key)
}

// SourceBeansSHA reads beans_sha from testdata/golden/SOURCE.
func SourceBeansSHA() (string, error) {
	return SourceField("beans_sha")
}

// TraitsCase is one row of traits.jsonl (corpus seed → hue/tone/palette).
type TraitsCase struct {
	Seed     string  `json:"seed"`
	Hue      float64 `json:"hue"`
	Tone     float64 `json:"tone"`
	ToneName string  `json:"toneName"`
	Palette  struct {
		BG   string `json:"bg"`
		Head string `json:"head"`
		Eye  string `json:"eye"`
	} `json:"palette"`
}

// LoadTraits parses testdata/golden/traits.jsonl.
func LoadTraits() ([]TraitsCase, error) {
	raw, err := os.ReadFile(filepath.Join(Root(), "traits.jsonl"))
	if err != nil {
		return nil, err
	}
	var out []TraitsCase
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var c TraitsCase
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return nil, fmt.Errorf("golden: traits.jsonl: %w", err)
		}
		out = append(out, c)
	}
	return out, sc.Err()
}

// ColorOracleCase is one row of color_oracle.jsonl (MixHex / FadeHex / Tinted).
// Success rows set Hex or HeadOut/EyeOut; error rows set Error to "bad_t" or "bad_hex".
// Non-finite FadeHex t values are encoded as the strings "NaN", "+Inf", "-Inf"
// because JSON cannot represent them.
type ColorOracleCase struct {
	Fn      string          `json:"fn"`
	A       string          `json:"a,omitempty"`
	B       string          `json:"b,omitempty"`
	T       json.RawMessage `json:"t,omitempty"`
	Hex     string          `json:"hex,omitempty"`
	Seed    string          `json:"seed,omitempty"`
	Tint    string          `json:"tint,omitempty"`
	Head    string          `json:"head,omitempty"`
	Eye     string          `json:"eye,omitempty"`
	HeadOut string          `json:"headOut,omitempty"`
	EyeOut  string          `json:"eyeOut,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// ParseT decodes the t field: a JSON number, or "NaN" / "+Inf" / "-Inf".
func (c ColorOracleCase) ParseT() (float64, error) {
	if len(c.T) == 0 {
		return 0, fmt.Errorf("golden: color oracle missing t")
	}
	if c.T[0] == '"' {
		var s string
		if err := json.Unmarshal(c.T, &s); err != nil {
			return 0, err
		}
		switch s {
		case "NaN":
			return math.NaN(), nil
		case "+Inf":
			return math.Inf(1), nil
		case "-Inf":
			return math.Inf(-1), nil
		default:
			return 0, fmt.Errorf("golden: unknown t string %q", s)
		}
	}
	var f float64
	if err := json.Unmarshal(c.T, &f); err != nil {
		return 0, err
	}
	return f, nil
}

// LoadColorOracle parses testdata/golden/color_oracle.jsonl.
func LoadColorOracle() ([]ColorOracleCase, error) {
	raw, err := os.ReadFile(filepath.Join(Root(), "color_oracle.jsonl"))
	if err != nil {
		return nil, err
	}
	var out []ColorOracleCase
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var c ColorOracleCase
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return nil, fmt.Errorf("golden: color_oracle.jsonl: %w", err)
		}
		out = append(out, c)
	}
	return out, sc.Err()
}
