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
