package blobatar

// Test hooks for hash_test.go / color_test.go (same-module white-box via export_test).

func SeedStateForTest(seed string) uint32 { return seedState(seed, true) }

func StreamForTest(state uint32, key string) float64 { return stream(state, key) }

func ShapeForSeed(seed string) string { return shapeForSeed(seed) }

func InGamutForTest(r, g, b float64) bool { return inGamut([3]float64{r, g, b}) }
