/**
 * Compare the complete expression oracle. Only fitted numbers under the root
 * `eyes` output in expressions.jsonl/custom.jsonl use absolute epsilon 1e-9.
 * Authored poses, interpolations, inputs, strings, keys/order and files stay exact.
 *
 * Usage: bun tools/gen-fixtures/compare-expression.ts <committed-dir> <regenerated-dir>
 */
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

const EPS = 1e-9;
const FILES = ["custom.jsonl", "expressions.jsonl", "lerp.jsonl", "rounding.jsonl", "states.jsonl"] as const;
const EYE_FILES = new Set<string>(["custom.jsonl", "expressions.jsonl"]);

function fail(message: string): never { throw new Error(message); }

// JSON.parse drops duplicate members and normalizes integer-like key order.
// On already-valid JSON, retain the lexical key sequences before either can
// hide a fixture change. Escaped key spellings are decoded for uniqueness.
function objectKeyOrder(text: string, label: string): string[][] {
  const order: string[][] = [];
  const stack: ({ kind: "object"; keys: string[]; seen: Set<string>; expectKey: boolean } | { kind: "array" })[] = [];
  const tokens = text.match(/"(?:\\.|[^"\\])*"|[{}\[\],]/g) ?? [];
  for (const token of tokens) {
    if (token === "{") {
      const keys: string[] = [];
      order.push(keys);
      stack.push({ kind: "object", keys, seen: new Set(), expectKey: true });
    } else if (token === "[") {
      stack.push({ kind: "array" });
    } else if (token === "}" || token === "]") {
      stack.pop();
    } else {
      const top = stack[stack.length - 1];
      if (top?.kind !== "object") continue;
      if (token === ",") top.expectKey = true;
      else if (top.expectKey) {
        const key = JSON.parse(token) as string;
        if (top.seen.has(key)) fail(`${label}: duplicate object key ${JSON.stringify(key)}`);
        top.keys.push(key);
        top.seen.add(key);
        top.expectKey = false;
      }
    }
  }
  return order;
}

function parseJSONL(text: string, label: string): { value: unknown; keyOrder: string[][] }[] {
  if (!text.endsWith("\n")) fail(`${label}: missing trailing LF`);
  return text.slice(0, -1).split("\n").map((line, i) => {
    let value: unknown;
    try { value = JSON.parse(line); }
    catch (error) { fail(`${label}:${i + 1}: invalid JSON: ${(error as Error).message}`); }
    return { value, keyOrder: objectKeyOrder(line, `${label}:${i + 1}`) };
  });
}

function typeOf(value: unknown): string {
  if (value === null) return "null";
  return Array.isArray(value) ? "array" : typeof value;
}

function compare(a: unknown, b: unknown, path: string, tolerance = false): void {
  const kind = typeOf(a);
  if (kind !== typeOf(b)) fail(`${path}: type ${kind} vs ${typeOf(b)}`);
  if (kind === "number") {
    const x = a as number, y = b as number;
    // JSON accepts exponents that overflow to Infinity. Check before equality.
    if (!Number.isFinite(x) || !Number.isFinite(y)) fail(`${path}: non-finite number ${x} vs ${y}`);
    if (tolerance ? Math.abs(x - y) > EPS : !Object.is(x, y)) {
      fail(`${path}: number ${x} vs ${y} (epsilon=${tolerance ? EPS : 0})`);
    }
    return;
  }
  if (kind === "string" || kind === "boolean" || kind === "null") {
    if (a !== b) fail(`${path}: ${JSON.stringify(a)} vs ${JSON.stringify(b)}`);
    return;
  }
  if (kind === "array") {
    const aa = a as unknown[], bb = b as unknown[];
    if (aa.length !== bb.length) fail(`${path}: array length ${aa.length} vs ${bb.length}`);
    for (let i = 0; i < aa.length; i++) compare(aa[i], bb[i], `${path}[${i}]`, tolerance);
    return;
  }
  if (kind !== "object") fail(`${path}: unsupported type ${kind}`);
  const ao = a as Record<string, unknown>, bo = b as Record<string, unknown>;
  const ak = Object.keys(ao), bk = Object.keys(bo);
  if (ak.length !== bk.length || ak.some((key, i) => key !== bk[i])) {
    fail(`${path}: keys/order ${JSON.stringify(ak)} vs ${JSON.stringify(bk)}`);
  }
  for (const key of ak) compare(ao[key], bo[key], `${path}.${key}`, tolerance);
}

export function compareExpressionJSONL(file: string, committed: string, regenerated: string): number {
  if (!FILES.some(name => name === file)) fail(`unknown expression fixture ${file}`);
  const a = parseJSONL(committed, `committed ${file}`);
  const b = parseJSONL(regenerated, `regenerated ${file}`);
  if (a.length !== b.length) fail(`${file}: row count ${a.length} vs ${b.length}`);
  for (let i = 0; i < a.length; i++) {
    const path = `${file}:${i + 1}`;
    if (typeOf(a[i]!.value) !== "object" || typeOf(b[i]!.value) !== "object") fail(`${path}: expected an object row`);
    compare(a[i]!.keyOrder, b[i]!.keyOrder, `${path}.lexical keys/order`);
    const left = a[i]!.value as Record<string, unknown>, right = b[i]!.value as Record<string, unknown>;
    // Do not project known fields: added or renamed fields must be visible.
    compare(Object.keys(left), Object.keys(right), `${path}.keys`);
    for (const key of Object.keys(left)) {
      compare(left[key], right[key], `${path}.${key}`, EYE_FILES.has(file) && key === "eyes");
    }
  }
  return a.length;
}

export function compareExpressionDirectories(committed: string, regenerated: string): Record<string, number> {
  for (const dir of [committed, regenerated]) {
    const entries = readdirSync(dir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name));
    if (entries.length !== FILES.length || entries.some((entry, i) => entry.name !== FILES[i] || !entry.isFile())) {
      fail(`${dir}: expected only ${FILES.join(", ")}, found ${entries.map(entry => entry.name).join(", ")}`);
    }
  }
  return Object.fromEntries(FILES.map(file => [file, compareExpressionJSONL(
    file, readFileSync(join(committed, file), "utf8"), readFileSync(join(regenerated, file), "utf8"),
  )]));
}

if (import.meta.main) {
  try {
    const [committed, regenerated] = process.argv.slice(2);
    if (!committed || !regenerated || process.argv.length !== 4) fail("usage: compare-expression.ts <committed-dir> <regenerated-dir>");
    const counts = compareExpressionDirectories(committed, regenerated);
    for (const [file, count] of Object.entries(counts)) {
      console.log(`${file}: structural OK (${EYE_FILES.has(file) ? `eyes epsilon=${EPS}; other fields exact` : "all fields exact"}; ${count} rows)`);
    }
  } catch (error) {
    console.error((error as Error).message);
    process.exitCode = 1;
  }
}
