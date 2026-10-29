package blobatar_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ported Go sources must carry Alain00's MIT copyright line. NOTICE keeps the
// full MIT text; the repo LICENSE remains GPL-3.0.
var portedGoFiles = []string{
	"hash.go",
	"traits.go",
	"color.go",
}

const mitHeaderPrefix = `// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar`

func TestPortedFileMITHeaders(t *testing.T) {
	root := moduleRoot(t)
	for _, name := range portedGoFiles {
		path := filepath.Join(root, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(raw)
		if !strings.HasPrefix(text, mitHeaderPrefix) {
			t.Errorf("%s: missing MIT port header (Copyright (c) 2026 Alain)", name)
		}
		head := text
		if len(head) > 400 {
			head = head[:400]
		}
		if !strings.Contains(head, "MIT License") {
			t.Errorf("%s: header must mention MIT License", name)
		}
		if !strings.Contains(head, "See NOTICE for the full license text.") {
			t.Errorf("%s: header must point at NOTICE", name)
		}
	}
}

func TestNOTICEKeepsAlainMIT(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(moduleRoot(t), "NOTICE"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"Copyright (c) 2026 Alain",
		"MIT License",
		"Permission is hereby granted, free of charge",
		"THE SOFTWARE IS PROVIDED \"AS IS\"",
		"a7fd546ebede49d0a9fa638945b9e534489782a2",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("NOTICE missing %q", want)
		}
	}
}

func TestLICENSEIsGPL3(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(moduleRoot(t), "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "GNU GENERAL PUBLIC LICENSE") {
		t.Fatal("LICENSE must be GPL")
	}
	if !strings.Contains(text, "Version 3, 29 June 2007") {
		t.Fatal("LICENSE must be GPL-3.0")
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Tests run with package dir as cwd (module root for this package).
	return wd
}
