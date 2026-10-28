// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar (vendor/core/hash.ts), MIT License.
// See NOTICE for the full license text.

package blobatar

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

const sep byte = 0xff

// Normalize mirrors normalizeSeed: NFC, trim, lowercase.
// Inputs a human considers equal (precomposed vs decomposed é, mixed case,
// surrounding whitespace) hash equally.
func Normalize(seed string) string {
	return strings.ToLower(strings.TrimSpace(norm.NFC.String(seed)))
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
