// Generate expression oracles from the TypeScript pinned in testdata/golden/SOURCE.
// Run: bun tools/gen-fixtures/gen-expression.ts /path/to/pinned-beans /tmp/expression-fixtures
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve, join } from "node:path";

const [beans, output] = process.argv.slice(2);
if (!beans || !output) throw new Error("usage: gen-expression.ts <beans-dir> <output-dir>");
const pkg = resolve(beans, "packages/beans-blobatar");
const roster = await import(join(pkg, "vendor/core/expression.ts"));
const { style } = await import(join(pkg, "vendor/core/styles/blob.ts"));
const { traits } = await import(join(pkg, "vendor/core/traits.ts"));
const { superellipse } = await import(join(pkg, "vendor/core/shape.ts"));
const { palette } = await import(join(pkg, "vendor/core/color.ts"));
const { bakePose, poseTransforms, lerpPose, IDENT, r3 } = await import(join(pkg, "vendor/core/morph.ts"));
const { BOT_EXPRESSIONS, BOT_AVATAR_STATES, prepare } = await import(join(pkg, "appearance.ts"));
const { tinted } = await import(join(pkg, "vendor/core/render.ts"));
const { SEEDS } = await import(join(pkg, "test/golden/corpus.ts"));

const names: string[] = [...BOT_EXPRESSIONS];
const cases: any[] = SEEDS.map((seed: string, i: number) => ({
  name: `corpus-${i}`, seed, expression: names[i % names.length]!, overrides: {},
}));
for (const [i, shape] of [0.11, 0.35, 0.54, 0.65, 0.745, 0.825, 0.8875, 0.9325, 0.965, 0.99].entries()) {
  for (const expression of names) cases.push({ name: `shape-${i}-${expression}`, seed: "expression-shapes", expression, overrides: { shape } });
}
for (const expression of ["mad", "love", "shy", "sick"]) {
  for (const [i, inputPalette] of [
    { head: "#d0d897", eye: "#0f1006", bg: "#AaBbCc" },
    { head: "#060606", eye: "#FFFFFF", bg: "#123456" },
    { head: "#FFFFFF", eye: "#000000", bg: "#FEDCBA" },
  ].entries()) cases.push({ name: `palette-${expression}-${i}`, seed: "palette-override", expression, overrides: {}, inputPalette });
}
const rows = cases.map((input: any) => {
  const t = traits(input.seed, true, input.overrides);
  const l = style.layout(t);
  const e = roster[input.expression];
  const pal = input.inputPalette ?? palette(t.num("hue", 0, 360), true, t("tone"));
  const baked = bakePose(l, e.p);
  return {
    ...input, pose: e.p, vars: e.vars(e.p), eyes: baked.l.eyes, wrap: baked.wrap,
    eyePaths: baked.l.eyes.map(superellipse),
    transforms: [-1, 0, 1].map((phase) => ({ phase, expected: poseTransforms(l, e.p, phase) })),
    palette: e.tint ? e.tint(pal, e.p) : pal,
  };
});
const interpolations = [];
for (const from of [null, ...names]) for (const to of [null, ...names]) for (const t of [-0.25, 0, 0.25, 0.5, 1, 1.25]) {
  interpolations.push({ from, to, t, expected: lerpPose(from ? roster[from].p : undefined, to ? roster[to].p : undefined, t) });
}
const custom = [
  { name: "partial-lock", p: { ...IDENT, esx: 1.45, esy: 0.22, tilt: -19.5, edy: 0.125, edx: -1.25, lock: 0.35, esx2: -0.1, esy2: 0.2, tilt2: 7.5, edy2: -1.1235, rock: 0.4, bdy: -1.2345 } },
  { name: "heat-only", p: { ...IDENT, heat: 0.65 } },
  { name: "signed-zero", p: { ...IDENT, bdy: -0, tilt: -0, edy: -0 } },
].map(({ name, p }) => {
  const l = style.layout(traits("custom-pose"));
  const baked = bakePose(l, p);
  return { name, seed: "custom-pose", pose: p, vars: roster.poseVars(p), eyes: baked.l.eyes, wrap: baked.wrap,
    eyePaths: baked.l.eyes.map(superellipse), transforms: [-1, 0, 1].map((phase) => ({ phase, expected: poseTransforms(l, p, phase) })) };
});
const states = [];
for (const state of BOT_AVATAR_STATES) for (const expression of names) {
  const look = { version: 1, base: { expression: "happy", palette: { head: "#123456", eye: "#FEDCBA", bg: "#ABCDEF" } }, states: { [state]: { expression } } };
  const seed = "state-expression";
  const result = prepare(seed, look, state);
  const pal = tinted(result.palette, result.expression);
  states.push({ state, seed, look, pose: result.expression?.p ?? IDENT,
    palette: { head: pal.head.toUpperCase(), eye: pal.eye.toUpperCase(), bg: pal.bg.toUpperCase() } });
}
mkdirSync(output, { recursive: true });
const save = (name: string, values: unknown[]) => writeFileSync(join(output, name), values.map((v) => JSON.stringify(v)).join("\n") + "\n");
save("expressions.jsonl", rows);
save("lerp.jsonl", interpolations);
save("custom.jsonl", custom);
save("states.jsonl", states);
save("rounding.jsonl", [-1.2345, -1.23450001, -0.0005, -0.00050001, -0, 0, 0.0005, 1.2345, 1.23449999].map((value) => ({ value, expected: r3(value) })));
