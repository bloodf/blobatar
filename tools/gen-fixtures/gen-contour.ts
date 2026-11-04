// Generate the synthetic contour fixtures from the pinned Beans TypeScript.
// Run: bun tools/gen-fixtures/gen-contour.ts /path/to/beans /tmp/contour-fixtures
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve, join } from "node:path";

const [beans, output] = process.argv.slice(2);
if (!beans || !output) throw new Error("usage: gen-contour.ts <beans-dir> <output-dir>");
const pkg = resolve(beans, "packages/beans-blobatar");
const { pathCubics, normalizeContour } = await import(join(pkg, "geometry.ts"));
const { superellipse, blobPath, polygon, box, taper } = await import(join(pkg, "vendor/core/shape.ts"));
const contours = [
  ["M 0 0 L 10 0 L 10 10 L 0 10 Z", 8, [5, 5]],
  ["M 0 0 Q 10 -5 10 0 C 15 5 15 10 10 10 H 0 V 0 Z", 9, [5, 5]],
  ["M0 0 V10 H10 V0 Z", 7, [5, 5]],
] as const;
const ellipses = [
  { cx: 50, cy: 50, rx: 50, ry: 50, n: 2, rot: 0 },
  { cx: 23.21, cy: 40.23, rx: 17.23, ry: 24.65, n: 4, rot: 32 },
  { cx: 0, cy: 0, rx: 8, ry: 12, n: 8, rot: -45 },
  { cx: -1.005, cy: -2.005, rx: 0, ry: 0, n: 2, rot: 0 },
];
const shapes = [
  { kind: "blob", args: [50, 51, 20, 23, [1, 0.8, 1.1, 0.9, 0.7], 23], expected: blobPath(50, 51, 20, 23, [1, 0.8, 1.1, 0.9, 0.7], 23) },
  { kind: "blob", args: [50, 50, 32.125, 27.255, [1, 1.125, 1.25, 1.375, 1.5, 1.625, 1.75, 1.875, 2], 190], expected: blobPath(50, 50, 32.125, 27.255, [1, 1.125, 1.25, 1.375, 1.5, 1.625, 1.75, 1.875, 2], 190) },
  { kind: "polygon", args: [50, 50, 32, 27, 3, 0.3, 0], expected: polygon({ cx: 50, cy: 50, rx: 32, ry: 27, sides: 3, round: 0.3, rot: 0 }) },
  { kind: "polygon", args: [23, 17, 8, 9, 6, 1, 32], expected: polygon({ cx: 23, cy: 17, rx: 8, ry: 9, sides: 6, round: 1, rot: 32 }) },
  { kind: "box", args: [50, 50, 22, 17], expected: box(50, 50, 22, 17) },
  { kind: "taper", args: [30, 40, 19, 25, 1.7], expected: taper(30, 40, 19, 25, 1.7) },
];
mkdirSync(output, { recursive: true });
const save = (name: string, values: unknown[]) => writeFileSync(join(output, name), values.map((v) => JSON.stringify(v)).join("\n") + "\n");
save("normalization.jsonl", contours.map(([path, count, center]) => ({ path, count, center, expected: normalizeContour(pathCubics(path), count, center) })));
save("superellipse.jsonl", ellipses.map((input) => ({ input, expected: superellipse(input) })));
save("shapes.jsonl", shapes);
