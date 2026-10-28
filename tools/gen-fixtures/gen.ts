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

type BotLook = { version: 1; base: Record<string, unknown>; states?: Record<string, Record<string, unknown>> };
const look = (base: Record<string, unknown>): BotLook => ({ version: 1, base });

const FRAME_TIMES = [0, 137, 1000, 4321] as const;
const FRAME_AMPS = [0, 1] as const;
const LERP_TS = [0, 0.5, 1] as const;

/** Composed U+00E9 vs decomposed e + U+0301; Normalize must collapse them. */
const NFC_COMPOSED = "caf\u00e9";
const NFC_DECOMPOSED = "cafe\u0301";
const LONG_SEED = "x".repeat(10_000);

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

const geoLines: GeoLine[] = [];
const frameLines: FrameLine[] = [];
const lerpLines: LerpLine[] = [];

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
}

// Edge-case seeds.
const edgeSeeds = [
  "",
  NFC_COMPOSED,
  NFC_DECOMPOSED,
  LONG_SEED,
];
for (const seed of edgeSeeds) {
  pushGeometry(seed, null, "idle");
  pushFrames(seed, null, "idle");
}

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
// frames.jsonl: avatarFrame rows first, then lerp rows (distinct shape via toLook).
writeJSONL(join(outDir, "frames.jsonl"), [...frameLines, ...lerpLines]);

// SOURCE metadata.
const beansSha = (
  await Bun.$`git -C ${resolve(beansDir)} rev-parse HEAD`.text()
).trim();
const upstreamSha = "a7fd546ebede49d0a9fa638945b9e534489782a2";
// Stable across machines; the beans-dir argument is whatever checkout the
// operator (or CI) pointed at for beans_sha.
const cmd = "bun tools/gen-fixtures/gen.ts <beans-dir>";
const source = [
  `beans_sha=${beansSha}`,
  `upstream_blobatar_sha=${upstreamSha}`,
  `generator=${cmd}`,
  "",
].join("\n");
writeFileSync(join(outDir, "SOURCE"), source, "utf8");

// Fingerprint for the run log (not written into fixtures).
const finger = (name: string) =>
  createHash("sha256").update(readFileSync(join(outDir, name))).digest("hex").slice(0, 16);
console.log(
  JSON.stringify(
    {
      beansSha,
      geometry: geoLines.length,
      frames: frameLines.length,
      lerps: lerpLines.length,
      hashes: {
        "gen2.txt": finger("gen2.txt"),
        "geometry.jsonl": finger("geometry.jsonl"),
        "frames.jsonl": finger("frames.jsonl"),
      },
    },
    null,
    2,
  ),
);
