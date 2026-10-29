package blobatar_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Exact MIT headers required on every Go file ported from Alain00/blobatar.
// NOTICE keeps the full MIT text; the repo LICENSE remains GPL-3.0.
var expectedMITHeaders = map[string]string{
	"hash.go": `// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/hash.ts), MIT License.
// See NOTICE for the full license text.
`,
	"traits.go": `// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/traits.ts and the shape band table
// in vendor/core/styles/blob.ts), MIT License. See NOTICE for the full license text.
`,
	"color.go": `// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/color.ts), MIT License.
// See NOTICE for the full license text.
`,
	"look.go": `// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar via Beans packages/beans-blobatar/appearance.ts
// (validateBotLook / resolveBotAppearance), MIT License. See NOTICE for the
// full license text. Seeded palette/hue/tone come from color.go / traits.go.
`,
}

// hasExactMITHeader reports whether text begins with the expected header block.
// A bare mention of "NOTICE" elsewhere in the file is not enough.
func hasExactMITHeader(text, want string) bool {
	return strings.HasPrefix(text, want)
}

func TestPortedFileMITHeaders(t *testing.T) {
	root := moduleRoot(t)
	for name, want := range expectedMITHeaders {
		path := filepath.Join(root, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !hasExactMITHeader(string(raw), want) {
			t.Errorf("%s: missing exact MIT port header referencing NOTICE", name)
		}
	}
}

func TestMITHeaderRejectsNoticeOnlyMention(t *testing.T) {
	// Would pass a loose "contains NOTICE" check, but must fail the exact prefix.
	fake := `// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar somewhere, MIT License.
// NOTICE is mentioned but this is not the real header for any ported file.

package blobatar
`
	if !strings.Contains(fake, "NOTICE") {
		t.Fatal("precondition: fake must contain NOTICE")
	}
	if !strings.Contains(fake, "MIT License") {
		t.Fatal("precondition: fake must contain MIT License")
	}
	for name, want := range expectedMITHeaders {
		if hasExactMITHeader(fake, want) {
			t.Errorf("NOTICE-only / wrong-path header must not satisfy %s", name)
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
