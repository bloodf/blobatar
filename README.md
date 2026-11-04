# blobatar

Deterministic blob avatars for Go, extracted from Beans (`@beans/blobatar`,
itself a vendored copy of [Alain00/blobatar](https://github.com/Alain00/blobatar)
2.7.0).

## Status

**Pre-v0.1.0.** Parity work is in progress. Beans will not consume this module
until **v0.1.0**.

**v0.1.0 scope:** static figures plus `Frame` and `Lerp`. React Native
idle/morph worklets are out of scope. `looks.json` lists BotLook enums
(generated from Beans TS via `tools/gen-fixtures`; TS remains source of truth
until the Beans swap).

### Layout foundation

The internal gen2 layout now covers all ten silhouettes, their safe face
regions, decorations, fitted eyes, and byte-exact core and eye paths.
`testdata/layout/layout.jsonl` is a separate TypeScript oracle generated from
the same Beans SHA and Bun version in `testdata/golden/SOURCE`; it covers the
1,000-seed corpus, every shape-band boundary, extreme trait overrides, and
candidate-list overrides. Numeric layout fields under `expected` use absolute
ε `1e-9`; path strings, metadata and overrides are exact. CI regenerates and
compares the complete oracle, including fields and ordering, and rejects
non-finite numbers. This allows platform-level floating-point differences
without changing the committed fixtures or weakening path parity.

The layout-to-contour tests also check core, extra, petals and eyes against all
1,049 rows of the original `geometry.jsonl`, without modifying that fixture.
This does **not** complete G3: the full Geometry API, background and expression
assembly, SVG renderer, Frame and Lerp remain to be implemented.

```bash
bun tools/gen-fixtures/gen-layout.ts /path/to/pinned-beans /tmp/layout-fixtures
bun tools/gen-fixtures/compare-layout.ts testdata/layout /tmp/layout-fixtures
bun test tools/gen-fixtures/compare-layout.test.ts
```

### Expression foundation

The internal expression layer covers all 14 frozen poses, static eye baking,
per-eye transforms (including seesaw phase), pose variables, pose interpolation
and the four palette tints. Its separate TypeScript oracle, generated from the
same pinned source, checks 1,152 layout/expression cases, 1,350 interpolations,
and all 98 expression/state combinations. Eye paths and transform strings are
byte-exact; fitted eye numbers use absolute ε `1e-9`. Pose values and palette
hex are exact. CI tolerates numeric differences only in the top-level `eyes`
outputs of `expressions.jsonl` and `custom.jsonl`. Other numbers, metadata,
field order, files, strings and interpolation results remain exact; non-finite
numbers are rejected everywhere. The unchanged geometry fixture also checks
pose and palette.

G4b remains open until the production Geometry/SVG path uses this layer.
Geometry must retain unposed eyes plus the separate pose, while static SVG
bakes the eye channels and wraps only the figure in the body translation.
Tinting uses the effective palette, including its overrides, and leaves the
background unchanged. Frame loops and full geometry interpolation are not
implemented by the pose helpers.

```bash
bun tools/gen-fixtures/gen-expression.ts /path/to/pinned-beans /tmp/expression-fixtures
bun tools/gen-fixtures/compare-expression.ts testdata/expressions /tmp/expression-fixtures
bun test tools/gen-fixtures/compare-expression.test.ts
```

## Parity contract

| Surface | Tolerance |
|---|---|
| SVG markup and SHA-256 tripwire hashes (`gen2.txt`) | byte-exact |
| Seed hashes (`Normalize` + murmur-style stream) | exact `uint32` / float64 |
| Traits / tone / hue / palette hex (`traits.jsonl`) | floats ε `1e-9`; hex exact |
| MixHex / FadeHex / Tinted (`color_oracle.jsonl`) | hex exact; error rows must error |
| Geometry (`geometry.jsonl`) | absolute ε `1e-9` |
| Frames and lerps (`frames.jsonl`) | absolute ε `1e-9` |

Golden fixtures under `testdata/golden/` are generated **from the Beans
TypeScript only**. They are **never regenerated to make a test pass**. A diff
means the Go port drifted, or a deliberate visual change needs a new generation
and CTO sign-off. See `testdata/golden/SOURCE` for the pinned Beans SHA, Bun
version (Beans CI uses `oven-sh/setup-bun@v2` unpinned; fixtures pin the
resolved version), and the exact generator command.

`gen2.txt` (and its hashes) are byte-exact. `geometry.jsonl` / `frames.jsonl`
are compared structurally with absolute ε `1e-9`. `normalize.jsonl` is the JS
oracle for `Normalize`.

`FromHex` accepts only `#RRGGBB` / `#rrggbb` (leading `#` required; exactly 6
hex digits after `#`; bare `rrggbb` is rejected — stricter than the TS
oracle's silent `parseInt`). It returns an error for empty input, `"#"`,
`"#12"`, `"#11223344"`, `"#zz"`, or non-hex characters, and never panics.
`MixHex`, `FadeHex`, and `Tinted` validate before slicing and propagate
errors. `FadeHex` also rejects non-finite `t` and `t` outside `[0, 1]`
(never malformed hex). `inGamut` returns false for NaN/±Inf. `ToHex`
returns an error for non-finite OKLCh components (never a silent
`#000000`, never `int(math.Round(NaN))`).

`TraitOverrides` is `map[string]TraitOverride` (`FixedTrait` / `ListTrait`);
JSON accepts only a number or number array. `Int` with `max < min` matches
Beans arithmetic (not an error).

### TraitOverrides migration

Callers that previously put bare numbers (or `[]float64`) in a
`map[string]any` / loose override map must construct typed values:

```go
// Before (no longer compiles):
overrides := map[string]any{
    "shape":   0.965,
    "eye.gap": []float64{0.1, 0.2},
}

// After:
overrides := blobatar.TraitOverrides{
    "shape":   blobatar.FixedTrait(0.965),
    "eye.gap": blobatar.ListTrait(0.1, 0.2),
}
```

JSON still accepts `number | number[]` only (`{"shape":0.965,"eye.gap":[0.1,0.2]}`).

A zero-value `TraitOverride{}` (for example `overrides["shape"] = blobatar.TraitOverride{}`)
is a **fixed override to 0**, not “leave this trait alone”. To leave a trait
unset, omit its key from the map. (`ListTrait()` with no candidates also falls
through to the hash, matching an omitted key.)

Authored tables (`hot`/`rose`/`blush`/`bile` tints, contrast floors, tone band
edges/names) are unexported; use `HotTint`, `RoseTint`, `BlushTint`, `BileTint`,
`TintTargets`, `ContrastFloors`, `ToneBandEdges`, and `ToneBandNames` (each
returns a copy).

```bash
bun tools/gen-fixtures/gen.ts /path/to/beans
bun tools/gen-fixtures/compare.ts testdata/golden /path/to/regenerated
```

## License

GPL-3.0. Upstream Alain00/blobatar is MIT; see `NOTICE` and the MIT header on
every Go file ported from the vendored TypeScript.
