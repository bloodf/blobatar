import { describe, expect, test } from "bun:test";
import { spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { compareLayoutDirectories, compareLayoutJSONL } from "./compare-layout";

const row = () => ({
  name: "band-0-below",
  seed: "layout-band",
  overrides: { shape: 0.22 - 1e-12, candidates: [0.1, 0.9] },
  expected: {
    shape: "round",
    body: { cx: 50, radii: [1, 1.1] },
    eyes: [{ rx: 2.6809931242190532, ry: 6.302691492012277 }],
    extra: ["M1 2L3 4Z"],
  },
  corePath: "M1 2C3 4 5 6 7 8Z",
  eyePaths: ["M2 3C4 5 6 7 8 9Z", "M3 4C5 6 7 8 9 10Z"],
});
const jsonl = (...rows: unknown[]) => rows.map(value => JSON.stringify(value)).join("\n") + "\n";
const cli = fileURLToPath(new URL("./compare-layout.ts", import.meta.url));

describe("layout numeric contract", () => {
  test("accepts the observed CI drift in eyes without rewriting fixtures", () => {
    const regenerated = row();
    regenerated.expected.eyes[0] = { rx: 2.6809931242190537, ry: 6.302691492012278 };
    expect(compareLayoutJSONL(jsonl(row()), jsonl(regenerated))).toBe(1);
  });

  test("uses absolute epsilon, including its boundary", () => {
    const committed = row();
    const regenerated = row();
    committed.expected.eyes[0]!.rx = 0;
    regenerated.expected.eyes[0]!.rx = 1e-9;
    expect(compareLayoutJSONL(jsonl(committed), jsonl(regenerated))).toBe(1);
    regenerated.expected.eyes[0]!.rx = 1.00001e-9;
    expect(() => compareLayoutJSONL(jsonl(committed), jsonl(regenerated))).toThrow("epsilon=1e-9");
    committed.expected.eyes[0]!.rx = 1e6;
    regenerated.expected.eyes[0]!.rx = 1e6 + 1e-7;
    expect(() => compareLayoutJSONL(jsonl(committed), jsonl(regenerated))).toThrow();
  });

  test("keeps scalar and candidate-list overrides exact below epsilon", () => {
    const regenerated = row();
    regenerated.overrides.shape += 1e-12;
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(regenerated))).toThrow("epsilon=0");
    const candidates = row();
    candidates.overrides.candidates[0]! += 1e-12;
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(candidates))).toThrow("epsilon=0");
  });

  test("does not give similarly named nested metadata numeric tolerance", () => {
    const a = { ...row(), metadata: { expected: { value: 0 } } };
    const b = { ...row(), metadata: { expected: { value: 1e-12 } } };
    expect(() => compareLayoutJSONL(jsonl(a), jsonl(b))).toThrow("epsilon=0");
  });

  test.each(["1e400", "-1e400"])("rejects non-finite %s, even when equal", value => {
    const text = jsonl(row()).replace('"cx":50', `"cx":${value}`);
    expect(() => compareLayoutJSONL(text, text)).toThrow("non-finite");
    expect(() => compareLayoutJSONL(jsonl(row()), text)).toThrow("non-finite");
    expect(() => compareLayoutJSONL(text, jsonl(row()))).toThrow("non-finite");
    const metadata = jsonl(row()).replace('"shape":0.219999999999', `"shape":${value}`);
    expect(() => compareLayoutJSONL(metadata, metadata)).toThrow("non-finite");
  });
});

describe("layout structure and exact values", () => {
  test.each(["corePath", "eyePaths", "extra"])("rejects a one-character %s change", field => {
    const regenerated = row();
    if (field === "corePath") regenerated.corePath += " ";
    if (field === "eyePaths") regenerated.eyePaths[0] += " ";
    if (field === "extra") regenerated.expected.extra[0] += " ";
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(regenerated))).toThrow();
  });

  test("rejects changed names, seeds, shape strings and numeric types", () => {
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl({ ...row(), name: "other" }))).toThrow();
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl({ ...row(), seed: "other" }))).toThrow();
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(row()).replace('"round"', '"organic"'))).toThrow();
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(row()).replace('"cx":50', '"cx":"50"'))).toThrow("type");
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(row()).replace('"cx":50', '"cx":null'))).toThrow("type");
  });

  test("rejects extra, missing and renamed fields at every depth", () => {
    const extraRoot = { ...row(), unexpected: true };
    const extraNested = row();
    Object.assign(extraNested.expected.eyes[0]!, { unexpected: true });
    const missing = jsonl(row()).replace(',"ry":6.302691492012277', "");
    const renamed = jsonl(row()).replace('"rx":', '"renamed":');
    for (const text of [jsonl(extraRoot), jsonl(extraNested), missing, renamed]) {
      expect(() => compareLayoutJSONL(jsonl(row()), text)).toThrow();
      expect(() => compareLayoutJSONL(text, jsonl(row()))).toThrow();
    }
  });

  test("rejects object key, array element and row reordering", () => {
    const reordered = Object.fromEntries(Object.entries(row()).reverse());
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(reordered))).toThrow();
    const nested = row();
    nested.expected.eyes[0] = { ry: 6.302691492012277, rx: 2.6809931242190532 };
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(nested))).toThrow("keys/order");
    const array = row();
    array.expected.body.radii.reverse();
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(array))).toThrow();
    const other = { ...row(), name: "other" };
    expect(() => compareLayoutJSONL(jsonl(row(), other), jsonl(other, row()))).toThrow();
  });

  test("rejects changed array lengths and row counts", () => {
    const regenerated = row();
    regenerated.eyePaths.pop();
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(regenerated))).toThrow("array length");
    expect(() => compareLayoutJSONL(jsonl(row()), jsonl(row(), row()))).toThrow("row count");
  });

  test.each(["", "\n", "{}", "not-json\n", "null\n", "[]\n", "1\n", '{"x":NaN}\n'])("rejects malformed input %j", text => {
    expect(() => compareLayoutJSONL(text, text)).toThrow();
  });
});

test("compares the complete dedicated directory, rejecting unexpected files", () => {
  const root = mkdtempSync(join(tmpdir(), "blobatar-layout-compare-"));
  try {
    const committed = join(root, "committed");
    const regenerated = join(root, "regenerated");
    mkdirSync(committed);
    mkdirSync(regenerated);
    writeFileSync(join(committed, "layout.jsonl"), jsonl(row()));
    writeFileSync(join(regenerated, "layout.jsonl"), jsonl(row()));
    expect(compareLayoutDirectories(committed, regenerated)).toBe(1);
    const valid = spawnSync(process.execPath, [cli, committed, regenerated], { encoding: "utf8" });
    expect(valid.status).toBe(0);
    expect(valid.stdout).toContain("structural OK");
    const invalidPath = row();
    invalidPath.corePath += " ";
    writeFileSync(join(regenerated, "layout.jsonl"), jsonl(invalidPath));
    const invalid = spawnSync(process.execPath, [cli, committed, regenerated], { encoding: "utf8" });
    expect(invalid.status).toBe(1);
    expect(invalid.stderr).toContain("corePath");
    writeFileSync(join(regenerated, "layout.jsonl"), jsonl(row()));
    writeFileSync(join(regenerated, "unexpected.jsonl"), "{}\n");
    expect(() => compareLayoutDirectories(committed, regenerated)).toThrow("expected only layout.jsonl");
    writeFileSync(join(committed, "unexpected.jsonl"), "{}\n");
    expect(() => compareLayoutDirectories(committed, regenerated)).toThrow("expected only layout.jsonl");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("CLI rejects missing arguments", () => {
  const result = spawnSync(process.execPath, [cli], { encoding: "utf8" });
  expect(result.status).toBe(1);
  expect(result.stderr).toContain("usage:");
});
