// Package golden reads the committed fixture files under testdata/golden.
package golden

import (
	"bufio"
	"bytes"
	"fmt"
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

// SourceBeansSHA reads beans_sha from testdata/golden/SOURCE.
func SourceBeansSHA() (string, error) {
	raw, err := os.ReadFile(filepath.Join(Root(), "SOURCE"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "beans_sha=") {
			return strings.TrimPrefix(line, "beans_sha="), nil
		}
	}
	return "", fmt.Errorf("golden: beans_sha missing from SOURCE")
}
