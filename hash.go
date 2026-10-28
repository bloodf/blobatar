// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/hash.ts), MIT License.
// See NOTICE for the full license text.

package blobatar

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

const sep byte = 0xff

// jsTrimRune reports whether r is stripped by String.prototype.trim:
// ECMAScript WhiteSpace + LineTerminator. Includes U+FEFF (BOM), U+00A0,
// U+1680, U+2000–U+200A, U+2028, U+2029, U+202F, U+205F, U+3000.
// Does NOT include U+0085 (NEL) or U+180E.
func jsTrimRune(r rune) bool {
	switch r {
	case 0x0009, // TAB
		0x000B, // VT
		0x000C, // FF
		0x0020, // SP
		0x00A0, // NBSP
		0x1680, // Ogham space mark
		0x2000, 0x2001, 0x2002, 0x2003, 0x2004,
		0x2005, 0x2006, 0x2007, 0x2008, 0x2009, 0x200A,
		0x202F, // narrow NBSP
		0x205F, // medium mathematical space
		0x3000, // ideographic space
		0xFEFF, // ZWNBSP / BOM
		0x000A, // LF
		0x000D, // CR
		0x2028, // line separator
		0x2029: // paragraph separator
		return true
	default:
		return false
	}
}

// Normalize mirrors normalizeSeed: NFC, then ES trim, then root-locale lower
// (cases.Lower(language.Und): Turkish İ → i̇, Final_Sigma ΟΔΟΣ → οδος).
// Inputs a human considers equal (precomposed vs decomposed é, mixed case,
// surrounding whitespace) hash equally.
//
// Invalid UTF-8: Go strings may hold raw invalid bytes that JS strings cannot.
// NFC, TrimFunc, and Lower leave those byte sequences unchanged and only
// transform well-formed runes; the returned string may still contain the
// original invalid bytes. (JS TextEncoder would emit U+FFFD for lone
// surrogates instead — that path does not arise for valid JS string seeds.)
//
// A fresh cases.Caser is created per call; x/text Casers are not safe for
// concurrent use when shared.
func Normalize(seed string) string {
	return cases.Lower(language.Und).String(strings.TrimFunc(norm.NFC.String(seed), jsTrimRune))
}

// jsStringLen returns the UTF-16 code unit length of s, matching JavaScript's
// String.length (surrogate pairs count as two).
func jsStringLen(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// feed mixes bytes into a 32-bit state using Math.imul / >>> semantics.
// uint32 multiply matches Math.imul's low-32-bit product.
func feed(h uint32, bytes []byte) uint32 {
	for _, b := range bytes {
		h = (h ^ uint32(b)) * 3432918353
		h = (h << 13) | (h >> 19)
	}
	return h
}

// finalize is murmur3 fmix32 — a bijection on uint32 with full avalanche.
func finalize(h uint32) uint32 {
	h = (h ^ (h >> 16)) * 2246822507
	h = (h ^ (h >> 13)) * 3266489909
	return h ^ (h >> 16)
}

// seedState hashes the seed once into a reusable state. Non-ASCII seeds are
// encoded to UTF-8 so hashing is over codepoints rather than UTF-16 units.
func seedState(seed string, normalize bool) uint32 {
	s := seed
	if normalize {
		s = Normalize(seed)
	}
	return feed(1779033703^uint32(jsStringLen(s)), []byte(s))
}

// stream derives one uniform float in [0, 1) for key, independent of every
// other key.
func stream(state uint32, key string) float64 {
	return float64(finalize(feed(feed(state, []byte{sep}), []byte(key)))) / 4294967296
}
