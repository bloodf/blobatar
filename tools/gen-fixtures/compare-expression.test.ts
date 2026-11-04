import { describe, expect, test } from "bun:test";
import { spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { compareExpressionDirectories, compareExpressionJSONL } from "./compare-expression";

const files = ["custom.jsonl", "expressions.jsonl", "lerp.jsonl", "rounding.jsonl", "states.jsonl"];
const row = () => ({
  name: "case", seed: "seed", expression: "thinking", overrides: { shape: 0.965, candidates: [0.1, 0.9] },
  pose: { esx: 1.15, edy2: -8.4, rock: 0.8 },
  vars: { "--mo-esx": "1.15" },
  eyes: [{ cx: 35, cy: 50, rx: 2.6809931242190532, ry: 6.302691492012277, n: 4, rot: 0 }],
  wrap: "translate(0 -0.4)", eyePaths: ["M1 2C3 4 5 6 7 8Z", "M2 3C4 5 6 7 8 9Z"],
  transforms: [{ phase: -1, expected: { eyes: ["translate(35 50)"], wrap: "translate(0 -0.4)" } }],
  palette: { bg: "#ABCDEF", head: "#123456", eye: "#FEDCBA" },
});
const jsonl = (...rows: unknown[]) => rows.map(v => JSON.stringify(v)).join("\n") + "\n";
const check = (a: unknown, b: unknown, file = "expressions.jsonl") => compareExpressionJSONL(file, jsonl(a), jsonl(b));
const cli = fileURLToPath(new URL("./compare-expression.ts", import.meta.url));

describe("expression numeric contract", () => {
  test.each(["expressions.jsonl", "custom.jsonl"])("accepts only fitted-eye numeric drift in %s", file => {
    const a = row(), b = row();
    b.eyes[0]!.rx = 2.6809931242190537;
    b.eyes[0]!.ry = 6.302691492012278;
    expect(check(a, b, file)).toBe(1);
    a.eyes[0]!.rx = 0;
    b.eyes[0]!.rx = 1e-9;
    expect(check(a, b, file)).toBe(1);
    b.eyes[0]!.rx = 1.0001e-9;
    expect(() => check(a, b, file)).toThrow("epsilon=1e-9");
    a.eyes[0]!.rx = 1e6; b.eyes[0]!.rx = 1e6 + 1e-7;
    expect(() => check(a, b, file)).toThrow();
  });

  test.each(["lerp.jsonl", "states.jsonl", "rounding.jsonl"])("keeps all numbers exact in %s", file => {
    const a = { t: 0.5, value: -1.2345, pose: { esx: 1.2 }, expected: { esx: 1.2 }, eyes: [{ rx: 1.2 }] };
    for (const changed of [
      { ...a, t: 0.5 + 1e-12 }, { ...a, value: -1.2345 + 1e-12 },
      { ...a, pose: { esx: 1.2 + 1e-12 } }, { ...a, expected: { esx: 1.2 + 1e-12 } },
      { ...a, eyes: [{ rx: 1.2 + 1e-12 }] },
    ]) expect(() => check(a, changed, file)).toThrow("epsilon=0");
  });

  test("keeps authored poses, overrides and phases exact below epsilon", () => {
    for (const change of [(b: ReturnType<typeof row>) => b.pose.esx += 1e-12,
      (b: ReturnType<typeof row>) => b.overrides.shape += 1e-12,
      (b: ReturnType<typeof row>) => b.overrides.candidates[0]! += 1e-12,
      (b: ReturnType<typeof row>) => b.transforms[0]!.phase += 1e-12]) {
      const b = row(); change(b);
      expect(() => check(row(), b)).toThrow("epsilon=0");
    }
    const a = { ...row(), metadata: { eyes: [{ rx: 1 }] } };
    const b = { ...row(), metadata: { eyes: [{ rx: 1 + 1e-12 }] } };
    expect(() => check(a, b)).toThrow("epsilon=0");
  });

  test.each(["1e400", "-1e400"])("rejects %s in tolerant and exact fields, even when equal", value => {
    for (const file of files) for (const text of [
      jsonl(row()).replace('"cx":35', `"cx":${value}`),
      jsonl(row()).replace('"esx":1.15', `"esx":${value}`),
    ]) {
      expect(() => compareExpressionJSONL(file, text, text)).toThrow("non-finite");
      expect(() => compareExpressionJSONL(file, jsonl(row()), text)).toThrow("non-finite");
      expect(() => compareExpressionJSONL(file, text, jsonl(row()))).toThrow("non-finite");
    }
  });
});

describe("expression exact strings and structure", () => {
  test("rejects changes in paths, transforms, palette, vars and metadata", () => {
    for (const change of [(b: ReturnType<typeof row>) => b.eyePaths[0] += " ",
      (b: ReturnType<typeof row>) => b.wrap += " ",
      (b: ReturnType<typeof row>) => b.transforms[0]!.expected.eyes[0] += " ",
      (b: ReturnType<typeof row>) => b.palette.bg = "#abcdef",
      (b: ReturnType<typeof row>) => b.vars["--mo-esx"] = "1.150",
      (b: ReturnType<typeof row>) => b.name = "other",
      (b: ReturnType<typeof row>) => b.seed = "other",
      (b: ReturnType<typeof row>) => b.expression = "idle"]) {
      const b = row(); change(b);
      expect(() => check(row(), b)).toThrow();
    }
  });

  test("rejects changed types, keys, field order, rows and array structure", () => {
    const original = jsonl(row());
    for (const changed of [
      original.replace('"cx":35', '"cx":"35"'), original.replace('"cx":35', '"cx":null'),
      original.replace('"cx":35,', ''), original.replace('"cx":', '"renamed":'),
      original.replace('"cx":35,', '"cx":35,"extra":1,'),
      original.replace('"cx":35,"cy":50', '"cy":50,"cx":35'),
      jsonl({ ...row(), unexpected: true }), jsonl(Object.fromEntries(Object.entries(row()).reverse())),
    ]) for (const [a, b] of [[original, changed], [changed, original]]) {
      expect(() => compareExpressionJSONL("expressions.jsonl", a!, b!)).toThrow();
    }
    const swapped = row(); swapped.eyePaths.reverse();
    expect(() => check(row(), swapped)).toThrow();
    const shorter = row(); shorter.eyePaths.pop();
    expect(() => check(row(), shorter)).toThrow("array length");
    const other = { ...row(), name: "other" };
    expect(() => compareExpressionJSONL("expressions.jsonl", jsonl(row(), other), jsonl(other, row()))).toThrow();
    expect(() => compareExpressionJSONL("expressions.jsonl", original, jsonl(row(), row()))).toThrow("row count");
  });

  test.each(["", "\n", "{}", "not-json\n", "null\n", "[]\n", "1\n", '{"x":NaN}\n'])("rejects malformed input %j", text => {
    expect(() => compareExpressionJSONL("expressions.jsonl", text, text)).toThrow();
  });

  test("rejects duplicate keys, including escaped aliases and hidden non-finite values", () => {
    const original = '{"expected":{"esx":1}}\n';
    for (const duplicate of [
      '{"expected":{"esx":1e400,"esx":1}}\n',
      '{"expected":{"esx":1,"esx":1}}\n',
      '{"expected":{"esx":1,"\\u0065sx":1}}\n',
      '{"expected":{"esx":1},"expected":{"esx":1}}\n',
    ]) {
      expect(() => compareExpressionJSONL("lerp.jsonl", original, duplicate)).toThrow("duplicate object key");
      expect(() => compareExpressionJSONL("lerp.jsonl", duplicate, original)).toThrow("duplicate object key");
      expect(() => compareExpressionJSONL("lerp.jsonl", duplicate, duplicate)).toThrow("duplicate object key");
    }
  });

  test("keeps lexical key ordering even for integer-like metadata keys", () => {
    const a = '{"metadata":{"2":2,"1":1}}\n';
    const b = '{"metadata":{"1":1,"2":2}}\n';
    expect(() => compareExpressionJSONL("expressions.jsonl", a, b)).toThrow("keys/order");
    expect(compareExpressionJSONL("expressions.jsonl", a, a)).toBe(1);
  });

  test("key scanning tolerates strings containing escaped punctuation", () => {
    const a = { quote: '\"{},[]', nested: [{ 'a\"\\b': "value,{}", "1": true }] };
    expect(check(a, a)).toBe(1);
  });

  test("rejects an unknown fixture name", () => {
    expect(() => check(row(), row(), "unknown.jsonl")).toThrow("unknown expression fixture");
  });
});

test("directory and CLI validate the complete file set", () => {
  const root = mkdtempSync(join(tmpdir(), "blobatar-expression-compare-"));
  try {
    const a = join(root, "a"), b = join(root, "b");
    mkdirSync(a); mkdirSync(b);
    for (const dir of [a, b]) for (const file of files) writeFileSync(join(dir, file), jsonl(row()));
    expect(Object.values(compareExpressionDirectories(a, b))).toEqual([1, 1, 1, 1, 1]);
    const valid = spawnSync(process.execPath, [cli, a, b], { encoding: "utf8" });
    expect(valid.status).toBe(0);
    expect(valid.stdout).toContain("eyes epsilon=1e-9");
    const changed = row(); changed.palette.eye = "#ffffff";
    writeFileSync(join(b, "expressions.jsonl"), jsonl(changed));
    const invalid = spawnSync(process.execPath, [cli, a, b], { encoding: "utf8" });
    expect(invalid.status).toBe(1);
    expect(invalid.stderr).toContain("palette.eye");
    writeFileSync(join(b, "expressions.jsonl"), jsonl(row()));
    writeFileSync(join(b, "unexpected.jsonl"), "{}\n");
    expect(() => compareExpressionDirectories(a, b)).toThrow("expected only");
    rmSync(join(b, "unexpected.jsonl"));
    rmSync(join(b, "states.jsonl"));
    expect(() => compareExpressionDirectories(a, b)).toThrow("expected only");
    mkdirSync(join(b, "states.jsonl"));
    expect(() => compareExpressionDirectories(a, b)).toThrow("expected only");
    writeFileSync(join(a, "unexpected.jsonl"), "{}\n");
    expect(() => compareExpressionDirectories(a, b)).toThrow("expected only");
  } finally { rmSync(root, { recursive: true, force: true }); }
});

test.each([{ args: [] }, { args: ["one"] }, { args: ["one", "two", "three"] }])("CLI rejects invalid arguments %j", ({ args }) => {
  const result = spawnSync(process.execPath, [cli, ...args], { encoding: "utf8" });
  expect(result.status).toBe(1);
  expect(result.stderr).toContain("usage:");
});
