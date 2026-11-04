// Generate layout oracles from the Beans TypeScript pinned in testdata/golden/SOURCE.
// Run: bun tools/gen-fixtures/gen-layout.ts /path/to/beans /tmp/layout-fixtures
// Existing golden fixtures are deliberately neither read as outputs nor rewritten.
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve, join } from "node:path";

const [beans, output] = process.argv.slice(2);
if (!beans || !output) throw new Error("usage: gen-layout.ts <beans-dir> <output-dir>");
const pkg = resolve(beans, "packages/beans-blobatar");
const { style } = await import(join(pkg, "vendor/core/styles/blob.ts"));
const { traits } = await import(join(pkg, "vendor/core/traits.ts"));
const { superellipse } = await import(join(pkg, "vendor/core/shape.ts"));
const { SEEDS } = await import(join(pkg, "test/golden/corpus.ts"));

type Overrides = Record<string, number | number[]>;
const cases: { name: string; seed: string; overrides?: Overrides }[] =
  SEEDS.map((seed: string, i: number) => ({ name: `corpus-${i}`, seed }));
const edges = [0.22, 0.48, 0.6, 0.7, 0.79, 0.86, 0.915, 0.95, 0.98, 1];
let lower = 0;
for (const [i, upper] of edges.entries()) {
  // Both sides and the exact edge freeze the exclusive-upper-bound selection.
  for (const [suffix, value] of [["below", upper - 1e-12], ["at", upper], ["above", upper + 1e-12]] as const)
    cases.push({ name: `band-${i}-${suffix}`, seed: "layout-band", overrides: { shape: value } });
  // Every layout-affecting trait is driven to either end of its allowed range.
  const keys = ["body.r", "body.x", "body.y", "body.ratio", "body.n", "body.rot", "body.pts",
    "capsule.squat", "poly.round", "droplet.tip", "nub.n", "cloud.n", "sun.n", "sun.dist", "sun.r", "sun.rot",
    "eye.rx", "eye.ratio", "eye.scale", "eye.stretch", "eye.gap", "gaze.x", "gaze.y", "eye.dy", "eye.lean", "eye.lean2", "eye.n",
    ...Array.from({ length: 8 }, (_, j) => `body.r${j}`), ...Array.from({ length: 6 }, (_, j) => `cloud.r${j}`),
    "nub.a0", "nub.a1", "nub.r0", "nub.r1"];
  for (const v of [0, 1]) {
    const overrides: Overrides = Object.fromEntries(keys.map((key) => [key, v]));
    overrides.shape = (lower + upper) / 2;
    cases.push({ name: `extreme-${i}-${v}`, seed: "layout-extreme", overrides });
  }
  lower = upper;
}
cases.push({ name: "list-overrides", seed: "layout-list", overrides: { shape: [0.1, 0.965, 0.99], "eye.gap": [0.1, 0.9], "body.r": [] } });
cases.push({ name: "empty-seed", seed: "" });
cases.push({ name: "normalized-seed", seed: "  CAFÉ\u0301  " });

const rows = cases.map((input) => {
  const l = style.layout(traits(input.seed, true, input.overrides));
  const { cx, cy, rx, ry } = l.face;
  return {
    ...input,
    expected: { shape: l.shape, body: l.body, face: { cx, cy, rx, ry }, petals: l.petals, extra: l.extra, eyes: l.eyes },
    corePath: l.draw ? l.draw(l.body) : superellipse(l.body),
    eyePaths: l.eyes.map(superellipse),
  };
});
mkdirSync(output, { recursive: true });
writeFileSync(join(output, "layout.jsonl"), rows.map((v) => JSON.stringify(v)).join("\n") + "\n");
