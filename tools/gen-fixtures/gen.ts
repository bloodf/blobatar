/**
 * Generate golden fixtures from a Beans checkout of packages/beans-blobatar.
 *
 * Usage: bun tools/gen-fixtures/gen.ts <beans-dir>
 *
 * Fixtures are never regenerated to make a test pass. A diff means the Go port
 * drifted, or a deliberate visual change needs a new generation and CTO sign-off.
 */

import { mkdirSync, copyFileSync, writeFileSync, readFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { createHash } from "node:crypto";

const beansDir = process.argv[2];
if (!beansDir) {
  console.error("usage: bun tools/gen-fixtures/gen.ts <beans-dir>");
  process.exit(1);
}

const pkg = resolve(beansDir, "packages/beans-blobatar");
const outDir = resolve(import.meta.dir, "../../testdata/golden");

const {
  botAvatarGeometry,
  avatarFrame,
  interpolateAvatarGeometry,
  BOT_SHAPES,
  BOT_EXPRESSIONS,
  BOT_TONES,
  BOT_BACKGROUNDS,
  BOT_AVATAR_STATES,
} = await import(join(pkg, "index.ts"));
const { SEEDS } = await import(join(pkg, "test/golden/corpus.ts"));
const { normalizeSeed } = await import(join(pkg, "vendor/core/hash.ts"));
const { traits } = await import(join(pkg, "vendor/core/traits.ts"));
const { palette } = await import(join(pkg, "vendor/core/color.ts"));

/** Upper edges of the tone swatches in vendor/core/color.ts; each band excludes its edge. */
const TONE_EDGES = [0.2, 0.36, 0.62, 0.8, 0.93, 1] as const;
const toneName = (v: number): string =>
  BOT_TONES[TONE_EDGES.findIndex((edge) => v < edge)] ?? "pastel";


type BotLook = { version: 1; base: Record<string, unknown>; states?: Record<string, Record<string, unknown>> };
const look = (base: Record<string, unknown>): BotLook => ({ version: 1, base });

const FRAME_TIMES = [0, 137, 1000, 4321] as const;
const FRAME_AMPS = [0, 1] as const;
const LERP_TS = [0, 0.5, 1] as const;

/** Composed U+00E9 vs decomposed e + U+0301; Normalize must collapse them. */
const NFC_COMPOSED = "caf\u00e9";
const NFC_DECOMPOSED = "cafe\u0301";
const LONG_SEED = "x".repeat(10_000);
/** Turkish capital I with dot; JS toLowerCase → i + combining dot above. */
const TURKISH_I = "\u0130";
/** Greek uppercase; Final_Sigma → οδος (ends with ς U+03C2). */
const FINAL_SIGMA = "\u039f\u0394\u039f\u03a3";
/** Leading BOM must be trimmed by Normalize. */
const LEADING_BOM = "\uFEFFblob";
/** Trailing NEL (U+0085) must be kept — not ES WhiteSpace. */
const TRAILING_NEL = "blob\u0085";

mkdirSync(outDir, { recursive: true });
copyFileSync(join(pkg, "test/golden/gen2.txt"), join(outDir, "gen2.txt"));

type GeoLine = {
  seed: string;
  look: BotLook | null;
  state: string;
  geometry: unknown;
};

type FrameLine = {
  seed: string;
  look: BotLook | null;
  state: string;
  t: number;
  amp: number;
  frame: unknown;
};

type LerpLine = {
  seed: string;
  look: BotLook;
  toLook: BotLook;
  state: string;
  t: number;
  geometry: unknown;
};

type NormalizeLine = {
  name: string;
  seed: string;
  normalized: string;
};

type TraitsLine = {
  seed: string;
  hue: number;
  tone: number;
  toneName: string;
  palette: { bg: string; head: string; eye: string };
};

const geoLines: GeoLine[] = [];
const frameLines: FrameLine[] = [];
const lerpLines: LerpLine[] = [];
const traitsLines: TraitsLine[] = [];

function pushGeometry(seed: string, l: BotLook | null, state: string) {
  const geometry = botAvatarGeometry(seed, l, state as never);
  geoLines.push({ seed, look: l, state, geometry });
}

function pushFrames(seed: string, l: BotLook | null, state: string) {
  const g = botAvatarGeometry(seed, l, state as never);
  for (const t of FRAME_TIMES) {
    for (const amp of FRAME_AMPS) {
      frameLines.push({ seed, look: l, state, t, amp, frame: avatarFrame(g, t, amp) });
    }
  }
}

// Corpus seeds (look null, idle).
for (const seed of SEEDS) {
  pushGeometry(seed, null, "idle");
  const t = traits(seed);
  const hue = t.num("hue", 0, 360);
  const tone = t("tone");
  const p = palette(hue, true, tone);
  traitsLines.push({
    seed,
    hue,
    tone,
    toneName: toneName(tone),
    palette: { bg: p.bg!, head: p.head!, eye: p.eye! },
  });
}

// Edge-case seeds (geometry + frames).
const edgeSeeds = [
  "",
  NFC_COMPOSED,
  NFC_DECOMPOSED,
  LONG_SEED,
  TURKISH_I,
  FINAL_SIGMA,
  LEADING_BOM,
  TRAILING_NEL,
];
for (const seed of edgeSeeds) {
  pushGeometry(seed, null, "idle");
  pushFrames(seed, null, "idle");
}

// Normalize oracle rows — generated from TS so Go is checked against JS output.
const normalizeCases: NormalizeLine[] = [
  { name: "empty", seed: "", normalized: normalizeSeed("") },
  { name: "ascii-trim-lower", seed: "  Alain@X.com  ", normalized: normalizeSeed("  Alain@X.com  ") },
  { name: "nfc-composed", seed: NFC_COMPOSED, normalized: normalizeSeed(NFC_COMPOSED) },
  { name: "nfc-decomposed", seed: NFC_DECOMPOSED, normalized: normalizeSeed(NFC_DECOMPOSED) },
  { name: "turkish-I", seed: TURKISH_I, normalized: normalizeSeed(TURKISH_I) },
  { name: "final-sigma", seed: FINAL_SIGMA, normalized: normalizeSeed(FINAL_SIGMA) },
  { name: "leading-BOM", seed: LEADING_BOM, normalized: normalizeSeed(LEADING_BOM) },
  { name: "trailing-NEL", seed: TRAILING_NEL, normalized: normalizeSeed(TRAILING_NEL) },
  { name: "NEL-both-sides", seed: "\u0085x\u0085", normalized: normalizeSeed("\u0085x\u0085") },
  { name: "NBSP-both-sides", seed: "\u00A0x\u00A0", normalized: normalizeSeed("\u00A0x\u00A0") },
  { name: "ideographic-space", seed: "\u3000x\u3000", normalized: normalizeSeed("\u3000x\u3000") },
  { name: "BOM-both-sides", seed: "\uFEFFx\uFEFF", normalized: normalizeSeed("\uFEFFx\uFEFF") },
];

// BotLook matrix: every shape / expression / tone / background / state once.
// UUID-style bot ids keep these distinct from the corpus templates.
let opt = 0;
const uuid = (n: number) =>
  `00000000-0000-4000-8000-${String(n).padStart(12, "0")}`;

for (const shape of BOT_SHAPES) {
  const id = uuid(1000 + opt++);
  const l = look({ shape });
  pushGeometry(id, l, "idle");
  pushFrames(id, l, "idle");
}
for (const expression of BOT_EXPRESSIONS) {
  const id = uuid(2000 + opt++);
  const l = look({ expression });
  pushGeometry(id, l, "idle");
  pushFrames(id, l, "idle");
}
for (const tone of BOT_TONES) {
  const id = uuid(3000 + opt++);
  const l = look({ tone });
  pushGeometry(id, l, "idle");
  pushFrames(id, l, "idle");
}
for (const background of BOT_BACKGROUNDS) {
  const id = uuid(4000 + opt++);
  const l = look({ background });
  pushGeometry(id, l, "idle");
  pushFrames(id, l, "idle");
}
for (const state of BOT_AVATAR_STATES) {
  const id = uuid(5000 + opt++);
  const l = look({});
  pushGeometry(id, l, state);
  pushFrames(id, l, state);
}

// Representative corpus seeds also get frames (keeps the file bounded).
for (const seed of SEEDS.slice(0, 12)) {
  pushFrames(seed, null, "idle");
}

// Lerp / morph samples at t = 0, 0.5, 1.
const lerpPairs: [string, BotLook, BotLook][] = [
  ["lerp-idle-mad", look({ expression: "idle" }), look({ expression: "mad" })],
  ["lerp-round-sun", look({ shape: "round" }), look({ shape: "sun" })],
  ["lerp-none-squircle", look({ background: "none" }), look({ background: "squircle" })],
];
for (const [seed, from, to] of lerpPairs) {
  const a = botAvatarGeometry(seed, from, "idle");
  const b = botAvatarGeometry(seed, to, "idle");
  for (const t of LERP_TS) {
    lerpLines.push({
      seed,
      look: from,
      toLook: to,
      state: "idle",
      t,
      geometry: interpolateAvatarGeometry(a, b, t),
    });
  }
}

function writeJSONL(path: string, rows: object[]) {
  // JSON.stringify insertion order; LF endings; no trailing spaces.
  const body = rows.map((r) => JSON.stringify(r)).join("\n") + "\n";
  writeFileSync(path, body, "utf8");
}

writeJSONL(join(outDir, "geometry.jsonl"), geoLines);
writeJSONL(join(outDir, "frames.jsonl"), [...frameLines, ...lerpLines]);
writeJSONL(join(outDir, "normalize.jsonl"), normalizeCases);
writeJSONL(join(outDir, "traits.jsonl"), traitsLines);

// SOURCE metadata. Bun version matches Beans CI's oven-sh/setup-bun@v2
// (unpinned → latest); we pin the resolved version that produced these fixtures.
const beansSha = (
  await Bun.$`git -C ${resolve(beansDir)} rev-parse HEAD`.text()
).trim();
const bunVersion = Bun.version; // e.g. 1.4.4
const upstreamSha = "a7fd546ebede49d0a9fa638945b9e534489782a2";
const cmd = "bun tools/gen-fixtures/gen.ts <beans-dir>";
const source = [
  `beans_sha=${beansSha}`,
  `upstream_blobatar_sha=${upstreamSha}`,
  `bun_version=${bunVersion}`,
  `generator=${cmd}`,
  "",
].join("\n");
writeFileSync(join(outDir, "SOURCE"), source, "utf8");

const finger = (name: string) =>
  createHash("sha256").update(readFileSync(join(outDir, name))).digest("hex").slice(0, 16);
console.log(
  JSON.stringify(
    {
      beansSha,
      bunVersion,
      geometry: geoLines.length,
      frames: frameLines.length,
      lerps: lerpLines.length,
      normalize: normalizeCases.length,
      traits: traitsLines.length,
      hashes: {
        "gen2.txt": finger("gen2.txt"),
        "geometry.jsonl": finger("geometry.jsonl"),
        "frames.jsonl": finger("frames.jsonl"),
        "normalize.jsonl": finger("normalize.jsonl"),
        "traits.jsonl": finger("traits.jsonl"),
      },
    },
    null,
    2,
  ),
);
