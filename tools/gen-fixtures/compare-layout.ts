/**
 * Compare layout oracles without depending on platform-specific float printing.
 * Only numbers below `expected` use absolute epsilon 1e-9. Inputs, strings,
 * object key order, array order and lengths, and fixture files remain exact.
 *
 * Usage: bun tools/gen-fixtures/compare-layout.ts <committed-dir> <regenerated-dir>
 */
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

const EPS = 1e-9;
const FILE = "layout.jsonl";

function fail(message: string): never {
  throw new Error(message);
}

function parseJSONL(text: string, label: string): unknown[] {
  if (!text.endsWith("\n")) fail(`${label}: missing trailing LF`);
  return text.slice(0, -1).split("\n").map((line, i) => {
    try {
      return JSON.parse(line);
    } catch (error) {
      fail(`${label}:${i + 1}: invalid JSON: ${(error as Error).message}`);
    }
  });
}

function typeOf(value: unknown): string {
  if (value === null) return "null";
  return Array.isArray(value) ? "array" : typeof value;
}

function compare(a: unknown, b: unknown, path: string, numericTolerance = false): void {
  const kind = typeOf(a);
  if (kind !== typeOf(b)) fail(`${path}: type ${kind} vs ${typeOf(b)}`);

  if (kind === "number") {
    const x = a as number;
    const y = b as number;
    // JSON.parse accepts e.g. 1e400 as Infinity. Check before equality so that
    // two identical non-finite values cannot silently satisfy the comparison.
    if (!Number.isFinite(x) || !Number.isFinite(y)) {
      fail(`${path}: non-finite number ${x} vs ${y}`);
    }
    if (numericTolerance ? Math.abs(x - y) > EPS : !Object.is(x, y)) {
      fail(`${path}: number ${x} vs ${y} (epsilon=${numericTolerance ? EPS : 0})`);
    }
    return;
  }
  if (kind === "string" || kind === "boolean" || kind === "null") {
    if (a !== b) fail(`${path}: ${JSON.stringify(a)} vs ${JSON.stringify(b)}`);
    return;
  }
  if (kind === "array") {
    const aa = a as unknown[];
    const bb = b as unknown[];
    if (aa.length !== bb.length) fail(`${path}: array length ${aa.length} vs ${bb.length}`);
    for (let i = 0; i < aa.length; i++) compare(aa[i], bb[i], `${path}[${i}]`, numericTolerance);
    return;
  }
  if (kind !== "object") fail(`${path}: unsupported type ${kind}`);
  const ao = a as Record<string, unknown>;
  const bo = b as Record<string, unknown>;
  const ak = Object.keys(ao);
  const bk = Object.keys(bo);
  if (ak.length !== bk.length || ak.some((key, i) => key !== bk[i])) {
    fail(`${path}: keys/order ${JSON.stringify(ak)} vs ${JSON.stringify(bk)}`);
  }
  for (const key of ak) compare(ao[key], bo[key], `${path}.${key}`, numericTolerance);
}

export function compareLayoutJSONL(committed: string, regenerated: string): number {
  const a = parseJSONL(committed, `committed ${FILE}`);
  const b = parseJSONL(regenerated, `regenerated ${FILE}`);
  if (a.length !== b.length) fail(`${FILE}: row count ${a.length} vs ${b.length}`);
  for (let i = 0; i < a.length; i++) {
    const path = `${FILE}:${i + 1}`;
    if (typeOf(a[i]) !== "object" || typeOf(b[i]) !== "object") {
      fail(`${path}: expected an object row`);
    }
    const left = a[i] as Record<string, unknown>;
    const right = b[i] as Record<string, unknown>;
    // Compare the complete row's key set/order before dispatching values. A
    // projection onto known fields would conceal newly added or renamed keys.
    compare(Object.keys(left), Object.keys(right), `${path}.keys`);
    for (const key of Object.keys(left)) {
      compare(left[key], right[key], `${path}.${key}`, key === "expected");
    }
  }
  return a.length;
}

export function compareLayoutDirectories(committed: string, regenerated: string): number {
  for (const dir of [committed, regenerated]) {
    const entries = readdirSync(dir, { withFileTypes: true });
    if (entries.length !== 1 || entries[0]!.name !== FILE || !entries[0]!.isFile()) {
      fail(`${dir}: expected only ${FILE}, found ${entries.map(entry => entry.name).join(", ")}`);
    }
  }
  return compareLayoutJSONL(
    readFileSync(join(committed, FILE), "utf8"),
    readFileSync(join(regenerated, FILE), "utf8"),
  );
}

if (import.meta.main) {
  try {
    const [committed, regenerated] = process.argv.slice(2);
    if (!committed || !regenerated || process.argv.length !== 4) {
      fail("usage: compare-layout.ts <committed-dir> <regenerated-dir>");
    }
    const count = compareLayoutDirectories(committed, regenerated);
    console.log(`${FILE}: structural OK (expected numbers epsilon=${EPS}; inputs and paths exact; ${count} rows)`);
  } catch (error) {
    console.error((error as Error).message);
    process.exitCode = 1;
  }
}
