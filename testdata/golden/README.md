# Golden fixtures

Pins (`beans_sha`, Bun, generator command) live in `SOURCE`. Regenerate with:

```bash
bun tools/gen-fixtures/gen.ts /path/to/beans
```

## gen2.txt provenance

`gen2.txt` is copied from Beans `packages/beans-blobatar/test/golden/gen2.txt`
by `tools/gen-fixtures/gen.ts` — it is **not** re-rendered here. The copy comes
from the Beans commit recorded as `beans_sha` in `SOURCE`
(`4760281f2b9cb24eec4438a473e7226cd4e3980c`). That Beans file is the frozen
Alain00/blobatar gen2 fixture (upstream `packages/blobatar/test/golden/gen2.txt`,
written with Alain00's `bun scripts/golden.ts --write`). Re-running the
generator only re-copies that path from a Beans checkout at `SOURCE`'s SHA.
