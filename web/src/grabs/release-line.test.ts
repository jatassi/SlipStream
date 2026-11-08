import { expect, test } from "vitest";

import type { GrabsListedRelease } from "../api/gen/model";
import { type RunNames, outcomeWords } from "./release-line";

const hd = "6f1d2c3b-4a5e-4f6a-8b7c-9d0e1f2a3b4c";
const uhd = "7a2e3d4c-5b6f-4a7b-9c8d-0e1f2a3b4c5d";

const names: RunNames = {
  targets: new Map([
    [hd, { name: "1080p", scores: { formats: [], increment: 1, minimum: 0, upgradeUntil: 0 } }],
    [uhd, { name: "2160p", scores: { formats: [], increment: 1, minimum: 0, upgradeUntil: 0 } }],
  ]),
  qualities: new Map(),
  formats: new Map(),
  grabbedFor: ["1080p", "2160p"],
  indexerAt: new Map([[0, "Fixture Site"]]),
};

const listed = (more: Partial<GrabsListedRelease>): GrabsListedRelease => ({
  position: 1,
  title: "Heat.1995.1080p.WEB-DL.DDP5.1-FLUX",
  indexerId: null,
  indexerName: "Fixture Mirror",
  size: 6_657_199_309,
  seeders: 12,
  outcome: "outranked",
  rank: 2,
  copyOf: null,
  covers: [],
  formatIds: [],
  scores: [],
  reasons: [],
  lostAt: null,
  words: "WEBDL-1080p · DD+ 5.1 · FLUX · 6.2 GB",
  ...more,
});

const score = { step: "score", targetId: hd, own: "10", ref: "35" } as const;

test.each<[string, GrabsListedRelease, string]>([
  ["grabbed", listed({ outcome: "grabbed", rank: 1 }), "Grabbed for 1080p and 2160p."],
  [
    "not_taken",
    listed({ outcome: "not_taken", rank: 1 }),
    "It ranked above the grabbed Release. Its Indexer stopped, reached its limit or dropped it.",
  ],
  [
    "outranked",
    listed({ outcome: "outranked", lostAt: score }),
    "Its score at 1080p is 10. The grabbed Release scores 35.",
  ],
  [
    "outranked, tying at every step",
    listed({ outcome: "outranked" }),
    "It ties the grabbed Release at every step.",
  ],
  [
    "kept",
    listed({ outcome: "kept", lostAt: score }),
    "Its score at 1080p is 10. The first-ranked Release scores 35.",
  ],
  [
    "kept, ranked first",
    listed({ outcome: "kept", rank: 1 }),
    "It ranked first. The search grabbed nothing.",
  ],
  [
    "refused",
    listed({
      outcome: "refused",
      rank: null,
      reasons: [{ code: "wrong_title", soft: true, value: "Heat (1986)" }],
    }),
    "An automatic search skips it. A Manual search can grab it.",
  ],
])("a %s Release says so in words", (_, release, words) => {
  expect(outcomeWords(release, names)).toBe(words);
});

test("a grabbed Release whose Grab is gone from the list still says the search grabbed it", () => {
  expect(outcomeWords(listed({ outcome: "grabbed" }), { ...names, grabbedFor: [] })).toBe(
    "This search grabbed it.",
  );
});
