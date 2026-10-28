# blobatar

Deterministic blob avatars for Go, extracted from Beans (`@beans/blobatar`,
itself a vendored copy of [Alain00/blobatar](https://github.com/Alain00/blobatar)
2.7.0).

## Status

**Pre-v0.1.0.** Parity work is in progress. Beans will not consume this module
until **v0.1.0**.

**v0.1.0 scope:** static figures plus `Frame` and `Lerp`. React Native
idle/morph worklets are out of scope. A `looks.json` for BotLook lists comes in
a later PR.

## Parity contract

| Surface | Tolerance |
|---|---|
| SVG markup and SHA-256 tripwire hashes (`gen2.txt`) | byte-exact |
| Seed hashes (`Normalize` + murmur-style stream) | exact `uint32` / float64 |
| Geometry (`geometry.jsonl`) | absolute ε `1e-9` |
| Frames and lerps (`frames.jsonl`) | absolute ε `1e-9` |

Golden fixtures under `testdata/golden/` are generated **from the Beans
TypeScript only**. They are **never regenerated to make a test pass**. A diff
means the Go port drifted, or a deliberate visual change needs a new generation
and CTO sign-off. See `testdata/golden/SOURCE` for the pinned Beans SHA and the
exact generator command.

```bash
bun tools/gen-fixtures/gen.ts /path/to/beans
```

## License

GPL-3.0. Upstream Alain00/blobatar is MIT; see `NOTICE` and the MIT header on
every Go file ported from the vendored TypeScript.
