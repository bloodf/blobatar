/**
 * Compare committed golden fixtures against a regeneration.
 *
 * - gen2.txt, normalize.jsonl: byte-exact
 * - geometry.jsonl / frames.jsonl: structural equality with absolute ε 1e-9
 *   on numbers; keys, types, row counts and non-numeric values must match exactly.
 *
 * Usage: bun tools/gen-fixtures/compare.ts <committed-dir> <regenerated-dir>
 */

import { readFileSync } from "node:fs";
import { join } from "node:path";

const EPS = 1e-9;

const committedDir = process.argv[2];
const regeneratedDir = process.argv[3];
if (!committedDir || !regeneratedDir) {
  console.error("usage: bun tools/gen-fixtures/compare.ts <committed-dir> <regenerated-dir>");
  process.exit(1);
}

function fail(msg: string): never {
  console.error(msg);
  process.exit(1);
}

function read(dir: string, name: string): Buffer {
  return readFileSync(join(dir, name));
}

function compareBytes(name: string): void {
  const a = read(committedDir, name);
  const b = read(regeneratedDir, name);
  if (!a.equals(b)) {
    fail(`${name}: byte mismatch (committed ${a.length} B vs regenerated ${b.length} B)`);
  }
  console.log(`${name}: byte-exact OK`);
}

compareBytes("gen2.txt");
compareBytes("normalize.jsonl");

function parseJSONL(buf: Buffer, label: string): unknown[] {
  const text = buf.toString("utf8");
  if (!text.endsWith("\n") && text.length > 0) {
    fail(`${label}: missing trailing LF`);
  }
  const lines = text === "" ? [] : text.replace(/\n$/, "").split("\n");
  return lines.map((line, i) => {
    try {
      return JSON.parse(line);
    } catch (e) {
      fail(`${label}:${i + 1}: invalid JSON: ${(e as Error).message}`);
    }
  });
}

function typeOf(v: unknown): string {
  if (v === null) return "null";
  if (Array.isArray(v)) return "array";
  return typeof v;
}

function compare(a: unknown, b: unknown, path: string): void {
  const ta = typeOf(a);
  const tb = typeOf(b);
  if (ta !== tb) {
    fail(`${path}: type ${ta} vs ${tb}`);
  }
  if (ta === "number") {
    const x = a as number;
    const y = b as number;
    if (Number.isNaN(x) && Number.isNaN(y)) return;
    if (x === y) return;
    if (!(Number.isFinite(x) && Number.isFinite(y)) || Math.abs(x - y) > EPS) {
      fail(`${path}: number ${x} vs ${y} (Δ=${Math.abs(x - y)}, eps=${EPS})`);
    }
    return;
  }
  if (ta === "string" || ta === "boolean" || ta === "null") {
    if (a !== b) fail(`${path}: ${JSON.stringify(a)} vs ${JSON.stringify(b)}`);
    return;
  }
  if (ta === "array") {
    const aa = a as unknown[];
    const bb = b as unknown[];
    if (aa.length !== bb.length) {
      fail(`${path}: array length ${aa.length} vs ${bb.length}`);
    }
    for (let i = 0; i < aa.length; i++) compare(aa[i], bb[i], `${path}[${i}]`);
    return;
  }
  // object
  const ao = a as Record<string, unknown>;
  const bo = b as Record<string, unknown>;
  const ak = Object.keys(ao);
  const bk = Object.keys(bo);
  if (ak.length !== bk.length) {
    fail(`${path}: key count ${ak.length} vs ${bk.length} (${ak.join(",")} vs ${bk.join(",")})`);
  }
  for (let i = 0; i < ak.length; i++) {
    if (ak[i] !== bk[i]) {
      fail(`${path}: key order/set diverge at ${i}: ${ak[i]} vs ${bk[i]}`);
    }
  }
  for (const k of ak) compare(ao[k], bo[k], `${path}.${k}`);
}

function compareJSONL(name: string): void {
  const a = parseJSONL(read(committedDir, name), `committed ${name}`);
  const b = parseJSONL(read(regeneratedDir, name), `regenerated ${name}`);
  if (a.length !== b.length) {
    fail(`${name}: row count ${a.length} vs ${b.length}`);
  }
  for (let i = 0; i < a.length; i++) {
    compare(a[i], b[i], `${name}:${i + 1}`);
  }
  console.log(`${name}: structural OK (ε=${EPS}, ${a.length} rows)`);
}

compareJSONL("geometry.jsonl");
compareJSONL("frames.jsonl");
console.log("fixtures compare: OK");
