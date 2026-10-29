# Golden fixtures

Pins (`beans_sha`, Bun, generator command) live in `SOURCE`. Regenerate with:

```bash
bun tools/gen-fixtures/gen.ts /path/to/beans
```

Also writes package-root `looks.json` (BotLook enum lists) and
`appearance_resolve.jsonl` (ResolveAppearance oracle from TS).

## gen2.txt provenance

`gen2.txt` is copied from Beans `packages/beans-blobatar/test/golden/gen2.txt`
by `tools/gen-fixtures/gen.ts` — it is **not** re-rendered here. The copy comes
from the Beans commit recorded as `beans_sha` in `SOURCE`
(`4760281f2b9cb24eec4438a473e7226cd4e3980c`), itself tracking Alain00/blobatar at
`upstream_blobatar_sha` (`a7fd546ebede49d0a9fa638945b9e534489782a2`). That Beans
file is the frozen Alain00/blobatar gen2 fixture (upstream
`packages/blobatar/test/golden/gen2.txt`, written with Alain00's
`bun scripts/golden.ts --write`). Re-running the generator only re-copies that
path from a Beans checkout at `SOURCE`'s SHA.
