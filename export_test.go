package blobatar

// Test hooks for hash_test.go (same-module white-box via export_test).

func SeedStateForTest(seed string) uint32 { return seedState(seed, true) }

func StreamForTest(state uint32, key string) float64 { return stream(state, key) }

func ShapeForSeed(seed string) string { return shapeForSeed(seed) }
