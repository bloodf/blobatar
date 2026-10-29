// Copyright (c) 2026 Alain
//
// Ported from Alain00/blobatar via Beans packages/beans-blobatar/appearance.ts
// (validateBotLook / resolveBotAppearance) and the palette ramp in
// vendor/core/color.ts, MIT License. See NOTICE for the full license text.
//
// MERGE(G2): BotPalette / seeded palette construction below are temporary
// stand-ins so ResolveAppearance can land without color.go. When the G2
// traits+palette PR merges, replace seedPalette and the oklch helpers with
// the shared color package types and delete the note.

package blobatar

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// BotLook schema version accepted by ValidateLook.
const lookVersion = 1

// Appearance is one sparse BotAppearance (base or per-state override).
type Appearance struct {
	Shape      string   // BotShape; empty = unset
	Expression string   // BotExpression; empty = unset
	Background string   // BotBackground; empty = unset
	Hue        *float64 // [0, 360); nil = unset
	Tone       string   // BotTone; empty = unset
	Palette    *BotPalette
	Motion     *bool // nil = unset
}

// BotPalette holds optional uppercase #RRGGBB channels.
// MERGE(G2): align with color.go Palette when that lands.
type BotPalette struct {
	Head string // empty = unset
	Eye  string // empty = unset
	Bg   string // empty = unset
}

// Look is a versioned BotLook document.
type Look struct {
	Version int
	Base    Appearance
	States  map[string]Appearance // nil if omitted
}

// ResolvedAppearance is the fully filled appearance ResolveAppearance returns.
type ResolvedAppearance struct {
	Shape      string
	Expression string
	Background string
	Hue        float64
	Tone       string
	Palette    BotPalette // all three channels set, uppercase #RRGGBB
	Motion     bool
}

// Enum lists — kept in sync with looks.json (verified by TestLookLooksJSON).
var (
	avatarStates = []string{"idle", "thinking", "responding", "working", "waiting", "retry", "error"}
	shapes       = []string{"round", "organic", "boxy", "capsule", "nub", "cloud", "droplet", "hexagon", "sun", "triangle"}
	expressions  = []string{"idle", "happy", "sad", "mad", "surprised", "wink", "sleepy", "smug", "unsure", "scared", "love", "shy", "sick", "thinking"}
	backgrounds  = []string{"none", "square", "circle", "squircle"}
	tones        = []string{"pastel", "pale", "mid", "deep", "bright", "ink"}
)

// Tone swatch positions used when a look names a tone (options() in appearance.ts).
var tonePosition = map[string]float64{
	"pastel": 0.1, "pale": 0.28, "mid": 0.49, "deep": 0.71, "bright": 0.865, "ink": 0.965,
}

// Upper edges of the tone swatches; each band excludes its edge.
var toneEdges = []float64{0.2, 0.36, 0.62, 0.8, 0.93, 1}

// hexColorRE matches uppercase #RRGGBB. Hand-rolled matcher, not a regexp.
var hexColorRE hexMatcher

// hexMatcher avoids importing regexp just for one anchored pattern.
type hexMatcher struct{}

func (hexMatcher) match(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for i := 1; i < 7; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func oneOf(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func toneName(v float64) string {
	for i, edge := range toneEdges {
		if v < edge {
			return tones[i]
		}
	}
	return tones[0]
}

type lookError string

func (e lookError) Error() string { return string(e) }

func failLook(path, message string) error {
	return lookError(path + " " + message)
}

func isRecord(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func onlyKeys(o map[string]any, allowed []string, path string) error {
	allow := make(map[string]struct{}, len(allowed))
	for _, k := range allowed {
		allow[k] = struct{}{}
	}
	for k := range o {
		if _, ok := allow[k]; !ok {
			return failLook(path, fmt.Sprintf("has unknown field %q", k))
		}
	}
	return nil
}

func oneOfString(v any, list []string, path string) (string, error) {
	s, ok := v.(string)
	if !ok || !oneOf(s, list) {
		return "", failLook(path, "must be one of "+strings.Join(list, ", "))
	}
	return s, nil
}

func parseAppearance(v any, path string) (Appearance, error) {
	o, ok := isRecord(v)
	if !ok {
		return Appearance{}, failLook(path, "must be an object")
	}
	if err := onlyKeys(o, []string{"shape", "expression", "background", "hue", "tone", "palette", "motion"}, path); err != nil {
		return Appearance{}, err
	}
	var out Appearance
	if raw, has := o["shape"]; has {
		s, err := oneOfString(raw, shapes, path+".shape")
		if err != nil {
			return Appearance{}, err
		}
		out.Shape = s
	}
	if raw, has := o["expression"]; has {
		s, err := oneOfString(raw, expressions, path+".expression")
		if err != nil {
			return Appearance{}, err
		}
		out.Expression = s
	}
	if raw, has := o["background"]; has {
		s, err := oneOfString(raw, backgrounds, path+".background")
		if err != nil {
			return Appearance{}, err
		}
		out.Background = s
	}
	if raw, has := o["hue"]; has {
		h, ok := asFloat64(raw)
		if !ok {
			return Appearance{}, failLook(path+".hue", "must be a finite number in [0, 360)")
		}
		if err := checkHueValue(h, path+".hue"); err != nil {
			return Appearance{}, err
		}
		out.Hue = &h
	}
	if raw, has := o["tone"]; has {
		s, err := oneOfString(raw, tones, path+".tone")
		if err != nil {
			return Appearance{}, err
		}
		out.Tone = s
	}
	if raw, has := o["palette"]; has {
		p, ok := isRecord(raw)
		if !ok {
			return Appearance{}, failLook(path+".palette", "must be an object")
		}
		if err := onlyKeys(p, []string{"head", "eye", "bg"}, path+".palette"); err != nil {
			return Appearance{}, err
		}
		pal := &BotPalette{}
		for _, k := range []string{"head", "eye", "bg"} {
			c, has := p[k]
			if !has {
				continue
			}
			cs, ok := c.(string)
			if !ok || !hexColorRE.match(cs) {
				return Appearance{}, failLook(path+".palette."+k, "must be uppercase #RRGGBB")
			}
			switch k {
			case "head":
				pal.Head = cs
			case "eye":
				pal.Eye = cs
			case "bg":
				pal.Bg = cs
			}
		}
		out.Palette = pal
	}
	if raw, has := o["motion"]; has {
		m, ok := raw.(bool)
		if !ok {
			return Appearance{}, failLook(path+".motion", "must be a boolean")
		}
		out.Motion = &m
	}
	return out, nil
}

func isFinite(h float64) bool {
	return !math.IsNaN(h) && !math.IsInf(h, 0)
}

func formatFloatSpecial(h float64) string {
	switch {
	case math.IsNaN(h):
		return "NaN"
	case math.IsInf(h, 1):
		return "+Inf"
	case math.IsInf(h, -1):
		return "-Inf"
	default:
		return fmt.Sprint(h)
	}
}

// checkHueValue requires a finite number in [0, 360). Non-finite values name
// the field path and the concrete value (NaN / +Inf / -Inf).
func checkHueValue(h float64, path string) error {
	if !isFinite(h) {
		return failLook(path, fmt.Sprintf("must be a finite number in [0, 360) (got %s)", formatFloatSpecial(h)))
	}
	if h < 0 || h >= 360 {
		return failLook(path, "must be a finite number in [0, 360)")
	}
	return nil
}

func asFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func versionIsOne(v any) bool {
	switch n := v.(type) {
	case float64:
		return n == 1
	case int:
		return n == 1
	case int64:
		return n == 1
	case json.Number:
		f, err := n.Float64()
		return err == nil && f == 1
	default:
		return false
	}
}

// ValidateLook is a strict structural check of an untrusted look; it returns a
// canonical copy. Accepts map[string]any (typical JSON), Look, or *Look.
// Typed Look values are validated directly (no JSON round-trip), so a NaN/Inf
// hue yields an error that names the field and the non-finite value.
func ValidateLook(input any) (Look, error) {
	switch v := input.(type) {
	case Look:
		return validateTypedLook(v)
	case *Look:
		if v == nil {
			return Look{}, failLook("look", "must be an object")
		}
		return validateTypedLook(*v)
	default:
		return validateMapLook(input)
	}
}

func validateMapLook(input any) (Look, error) {
	o, ok := isRecord(input)
	if !ok {
		return Look{}, failLook("look", "must be an object")
	}
	if err := onlyKeys(o, []string{"version", "base", "states"}, "look"); err != nil {
		return Look{}, err
	}
	if !versionIsOne(o["version"]) {
		return Look{}, failLook("look.version", "must be 1")
	}
	base, err := parseAppearance(o["base"], "look.base")
	if err != nil {
		return Look{}, err
	}
	look := Look{Version: lookVersion, Base: base}
	if raw, has := o["states"]; has {
		s, ok := isRecord(raw)
		if !ok {
			return Look{}, failLook("look.states", "must be an object")
		}
		if err := onlyKeys(s, avatarStates, "look.states"); err != nil {
			return Look{}, err
		}
		states := make(map[string]Appearance)
		for _, k := range avatarStates {
			if _, has := s[k]; !has {
				continue
			}
			a, err := parseAppearance(s[k], "look.states."+k)
			if err != nil {
				return Look{}, err
			}
			states[k] = a
		}
		look.States = states
	}
	return look, nil
}

func validateTypedLook(in Look) (Look, error) {
	if in.Version != lookVersion {
		return Look{}, failLook("look.version", "must be 1")
	}
	base, err := validateTypedAppearance(in.Base, "look.base")
	if err != nil {
		return Look{}, err
	}
	out := Look{Version: lookVersion, Base: base}
	if in.States == nil {
		return out, nil
	}
	states := make(map[string]Appearance, len(in.States))
	for k, a := range in.States {
		if !oneOf(k, avatarStates) {
			return Look{}, failLook("look.states", fmt.Sprintf("has unknown field %q", k))
		}
		ca, err := validateTypedAppearance(a, "look.states."+k)
		if err != nil {
			return Look{}, err
		}
		states[k] = ca
	}
	out.States = states
	return out, nil
}

func validateTypedAppearance(a Appearance, path string) (Appearance, error) {
	out := Appearance{}
	if a.Shape != "" {
		if !oneOf(a.Shape, shapes) {
			return Appearance{}, failLook(path+".shape", "must be one of "+strings.Join(shapes, ", "))
		}
		out.Shape = a.Shape
	}
	if a.Expression != "" {
		if !oneOf(a.Expression, expressions) {
			return Appearance{}, failLook(path+".expression", "must be one of "+strings.Join(expressions, ", "))
		}
		out.Expression = a.Expression
	}
	if a.Background != "" {
		if !oneOf(a.Background, backgrounds) {
			return Appearance{}, failLook(path+".background", "must be one of "+strings.Join(backgrounds, ", "))
		}
		out.Background = a.Background
	}
	if a.Hue != nil {
		if err := checkHueValue(*a.Hue, path+".hue"); err != nil {
			return Appearance{}, err
		}
		h := *a.Hue
		out.Hue = &h
	}
	if a.Tone != "" {
		if !oneOf(a.Tone, tones) {
			return Appearance{}, failLook(path+".tone", "must be one of "+strings.Join(tones, ", "))
		}
		out.Tone = a.Tone
	}
	if a.Palette != nil {
		pal := &BotPalette{}
		for _, ch := range []struct {
			name, val string
			set       *string
		}{
			{"head", a.Palette.Head, &pal.Head},
			{"eye", a.Palette.Eye, &pal.Eye},
			{"bg", a.Palette.Bg, &pal.Bg},
		} {
			if ch.val == "" {
				continue
			}
			if !hexColorRE.match(ch.val) {
				return Appearance{}, failLook(path+".palette."+ch.name, "must be uppercase #RRGGBB")
			}
			*ch.set = ch.val
		}
		out.Palette = pal
	}
	if a.Motion != nil {
		m := *a.Motion
		out.Motion = &m
	}
	return out, nil
}

// effective merges base with an optional state override; palette channels merge
// individually (base channel kept when the override omits it).
func effective(look Look, state string) Appearance {
	base := look.Base
	if state == "" || look.States == nil {
		return base
	}
	over, ok := look.States[state]
	if !ok {
		return base
	}
	merged := base
	if over.Shape != "" {
		merged.Shape = over.Shape
	}
	if over.Expression != "" {
		merged.Expression = over.Expression
	}
	if over.Background != "" {
		merged.Background = over.Background
	}
	if over.Hue != nil {
		h := *over.Hue
		merged.Hue = &h
	}
	if over.Tone != "" {
		merged.Tone = over.Tone
	}
	if over.Motion != nil {
		m := *over.Motion
		merged.Motion = &m
	}
	if base.Palette != nil || over.Palette != nil {
		merged.Palette = mergePalette(base.Palette, over.Palette)
	}
	return merged
}

func mergePalette(base, over *BotPalette) *BotPalette {
	out := &BotPalette{}
	if base != nil {
		*out = *base
	}
	if over == nil {
		return out
	}
	if over.Head != "" {
		out.Head = over.Head
	}
	if over.Eye != "" {
		out.Eye = over.Eye
	}
	if over.Bg != "" {
		out.Bg = over.Bg
	}
	return out
}

// ResolveAppearance merges look + state and fills seeded defaults for omitted
// identity fields. look may be nil (full seed defaults). Non-nil values are
// re-validated. state "" means no state override; unknown states error.
func ResolveAppearance(botID string, look any, state string) (ResolvedAppearance, error) {
	if state != "" && !oneOf(state, avatarStates) {
		return ResolvedAppearance{}, fmt.Errorf("unknown avatar state %q", state)
	}
	var app Appearance
	// Untyped nil and typed nil *Look both mean "no look" (seeded defaults).
	if look != nil && !isNilLookPointer(look) {
		checked, err := ValidateLook(look)
		if err != nil {
			return ResolvedAppearance{}, err
		}
		app = effective(checked, state)
	}
	return resolveFromAppearance(botID, app), nil
}

func isNilLookPointer(v any) bool {
	p, ok := v.(*Look)
	return ok && p == nil
}

func resolveFromAppearance(botID string, a Appearance) ResolvedAppearance {
	state := seedState(botID, true)
	hueUnit := stream(state, "hue")
	toneUnit := stream(state, "tone")

	hue := hueUnit * 360
	if a.Hue != nil {
		hue = *a.Hue
	}
	tone := toneName(toneUnit)
	tonePos := toneUnit
	if a.Tone != "" {
		tone = a.Tone
		tonePos = tonePosition[a.Tone]
	}

	shape := shapeForSeed(botID)
	if a.Shape != "" {
		shape = a.Shape
	}

	expr := "idle"
	if a.Expression != "" {
		expr = a.Expression
	}
	bg := "none"
	if a.Background != "" {
		bg = a.Background
	}
	motion := true
	if a.Motion != nil {
		motion = *a.Motion
	}

	seeded := seedPalette(hue, tonePos)
	pal := BotPalette{
		Head: strings.ToUpper(seeded.Head),
		Eye:  strings.ToUpper(seeded.Eye),
		Bg:   strings.ToUpper(seeded.Bg),
	}
	if a.Palette != nil {
		if a.Palette.Head != "" {
			pal.Head = a.Palette.Head
		}
		if a.Palette.Eye != "" {
			pal.Eye = a.Palette.Eye
		}
		if a.Palette.Bg != "" {
			pal.Bg = a.Palette.Bg
		}
	}

	return ResolvedAppearance{
		Shape:      shape,
		Expression: expr,
		Background: bg,
		Hue:        hue,
		Tone:       tone,
		Palette:    pal,
		Motion:     motion,
	}
}

// --- MERGE(G2): seeded palette (vendor/core/color.ts ramp) -----------------

type oklch struct {
	l, c, h float64
}

func toLinear(color oklch) [3]float64 {
	r := (color.h * math.Pi) / 180
	a := color.c * math.Cos(r)
	b := color.c * math.Sin(r)
	l_ := color.l + 0.3963377774*a + 0.2158037573*b
	m_ := color.l - 0.1055613458*a - 0.0638541728*b
	s_ := color.l - 0.0894841775*a - 1.291485548*b
	L := l_ * l_ * l_
	M := m_ * m_ * m_
	S := s_ * s_ * s_
	return [3]float64{
		4.0767416621*L - 3.3077115913*M + 0.2309699292*S,
		-1.2684380046*L + 2.6097574011*M - 0.3413193965*S,
		-0.0041960863*L - 0.7034186147*M + 1.707614701*S,
	}
}

func inGamut(rgb [3]float64) bool {
	for _, v := range rgb {
		if v < -1e-4 || v > 1+1e-4 {
			return false
		}
	}
	return true
}

func resolveOK(color oklch) [3]float64 {
	rgb := toLinear(color)
	if !inGamut(rgb) {
		lo, hi := 0.0, color.c
		for i := 0; i < 12; i++ {
			mid := (lo + hi) / 2
			if inGamut(toLinear(oklch{color.l, mid, color.h})) {
				lo = mid
			} else {
				hi = mid
			}
		}
		rgb = toLinear(oklch{color.l, lo, color.h})
	}
	for i := range rgb {
		if rgb[i] < 0 {
			rgb[i] = 0
		} else if rgb[i] > 1 {
			rgb[i] = 1
		}
	}
	return rgb
}

func luminance(color oklch) float64 {
	rgb := resolveOK(color)
	return 0.2126*rgb[0] + 0.7152*rgb[1] + 0.0722*rgb[2]
}

func contrastOK(a, b oklch) float64 {
	x, y := luminance(a), luminance(b)
	return (math.Max(x, y) + 0.05) / (math.Min(x, y) + 0.05)
}

func ensureContrast(fg, bg oklch, min float64) oklch {
	if contrastOK(fg, bg) >= min {
		return fg
	}
	lean := 1.0
	if fg.l < bg.l {
		lean = -1
	}
	for _, dir := range []float64{lean, -lean} {
		probe := fg
		for i := 0; i < 60; i++ {
			probe.l = math.Min(1, math.Max(0, probe.l+dir*0.02))
			if contrastOK(probe, bg) >= min {
				return probe
			}
			if probe.l == 0 || probe.l == 1 {
				break
			}
		}
	}
	black := oklch{0, 0, fg.h}
	white := oklch{1, 0, fg.h}
	if contrastOK(black, bg) >= contrastOK(white, bg) {
		return black
	}
	return white
}

func toHex(color oklch) string {
	rgb := resolveOK(color)
	out := make([]byte, 0, 7)
	out = append(out, '#')
	for _, v := range rgb {
		var s float64
		if v <= 0.0031308 {
			s = 12.92 * v
		} else {
			s = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		n := int(math.Round(s * 255))
		out = append(out, "0123456789abcdef"[n>>4], "0123456789abcdef"[n&15])
	}
	return string(out)
}

var toneSwatches = []struct {
	edge float64
	l, c float64
}{
	{0.2, 0.86, 0.085},
	{0.36, 0.9, 0.028},
	{0.62, 0.73, 0.135},
	{0.8, 0.62, 0.165},
	{0.93, 0.87, 0.16},
	{1.0, 0.34, 0.035},
}

func toneAt(v float64) (l, c float64) {
	for _, t := range toneSwatches {
		if v < t.edge {
			return t.l, t.c
		}
	}
	return toneSwatches[0].l, toneSwatches[0].c
}

var darkSurface = oklch{0.145, 0, 0}

const surfaceFloor = 1.5

func seedPalette(hue, tone float64) BotPalette {
	tl, tc := toneAt(tone)
	head := ensureContrast(oklch{tl, tc, hue}, darkSurface, surfaceFloor)
	eye := oklch{0.17, 0.02, hue}
	if head.l < 0.5 {
		eye = oklch{0.97, 0.012, hue}
	}
	bg := oklch{0.965, 0.01, hue}
	// FLOORS: head/bg 1.25, eye/head 4.5
	head = ensureContrast(head, bg, 1.25)
	eye = ensureContrast(eye, head, 4.5)
	return BotPalette{
		Head: toHex(head),
		Eye:  toHex(eye),
		Bg:   toHex(bg),
	}
}
