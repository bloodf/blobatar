package blobatar

// Test hooks for hash_test.go / color_test.go / look_test.go (same-module
// white-box via export_test).

func SeedStateForTest(seed string) uint32 { return seedState(seed, true) }

func StreamForTest(state uint32, key string) float64 { return stream(state, key) }

func ShapeForSeed(seed string) string { return shapeForSeed(seed) }

func InGamutForTest(r, g, b float64) bool { return inGamut([3]float64{r, g, b}) }

// Enum lists ValidateLook / ResolveAppearance use — compared to looks.json.
func AvatarStatesForTest() []string { return append([]string(nil), avatarStates...) }
func ShapesForTest() []string       { return append([]string(nil), shapes...) }
func ExpressionsForTest() []string  { return append([]string(nil), expressions...) }
func BackgroundsForTest() []string  { return append([]string(nil), backgrounds...) }
func BotTonesForTest() []string     { return append([]string(nil), botTones...) }

// ResolveFromAppearanceForTest exposes resolveFromAppearance for white-box
// coverage of the PaletteColors error path (unreachable via ValidateLook).
func ResolveFromAppearanceForTest(botID string, a Appearance) (ResolvedAppearance, error) {
	return resolveFromAppearance(botID, a)
}
