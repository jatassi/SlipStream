import { expect, test } from "vitest";
import { z } from "zod";

import filtersGolden from "../../../internal/targets/testdata/filters.json" with { type: "json" };
import upgradesGolden from "../../../internal/targets/testdata/upgrades.json" with { type: "json" };
import type { Layout, QualityKey } from "./order";
import {
  type Rules,
  type Side,
  type Why,
  judgeInWords,
  judgeMove,
  upgradeSentence,
} from "./upgrade-sentence";
import { attributes } from "./vocab";

const table = [
  "sdtv",
  "dvd",
  "webrip-480p",
  "hdtv-720p",
  "webrip-720p",
  "webdl-720p",
  "bluray-720p",
  "hdtv-1080p",
  "webrip-1080p",
  "webdl-1080p",
  "bluray-1080p",
  "remux-1080p",
  "hdtv-2160p",
  "webrip-2160p",
  "webdl-2160p",
  "bluray-2160p",
  "remux-2160p",
  "cam",
] as const satisfies readonly QualityKey[];

const key = z.enum(table);

const side = z.object({
  quality: key,
  version: z.number().default(1),
  real: z.number().default(0),
  repack: z.boolean().default(false),
  group: z.string().nullable().default(null),
  score: z.number().default(0),
});

const golden = z
  .object({
    targets: z.array(
      z.object({
        name: z.string(),
        tiers: z.array(z.array(key)),
        cutoff: key,
        scores: z.object({ minimum: z.number(), upgradeUntil: z.number(), increment: z.number() }),
        propers: z.enum(["prefer_and_upgrade", "prefer_no_upgrade", "dont_prefer"]),
        moves: z.array(
          z.object({
            from: side.nullable(),
            to: side,
            grabbable: z.boolean(),
            reasons: z.array(
              z.enum([
                "quality_not_in_target",
                "score_below_minimum",
                "not_an_upgrade",
                "repack_other_group",
              ]),
            ),
            rule: z.number(),
          }),
        ),
      }),
    ),
  })
  .parse(upgradesGolden);

const layoutOf = (tiers: QualityKey[][], cutoff: QualityKey): Layout => ({
  tiers,
  cutoff: tiers.findIndex((tier) => tier.includes(cutoff)),
});

const unscored = (layout: Layout): Rules => ({
  ...layout,
  scores: { minimum: 0, upgradeUntil: 0, increment: 1 },
  propers: "prefer_and_upgrade",
});

const rulesOf = ({ tiers, cutoff, scores, propers }: (typeof golden.targets)[number]): Rules => ({
  ...layoutOf(tiers, cutoff),
  scores,
  propers,
});

const refusalOf: { readonly [W in Exclude<Why, "not_in_target">]: readonly string[] } = {
  other_group: ["repack_other_group"],
  final: ["not_an_upgrade"],
  lower: ["not_an_upgrade"],
  older: ["not_an_upgrade"],
  tied: ["not_an_upgrade"],
  short: ["not_an_upgrade"],
  vacant: [],
  outside: [],
  proper: [],
  upgrade: [],
  gain: [],
};

const reasonsOf = ({ why, belowMinimum }: ReturnType<typeof judgeMove>): readonly string[] => {
  const score = belowMinimum ? ["score_below_minimum"] : [];
  return why === "not_in_target"
    ? ["quality_not_in_target", ...score]
    : [...score, ...refusalOf[why]];
};

const names: Readonly<Record<QualityKey, string>> = {
  sdtv: "SDTV",
  dvd: "DVD",
  "webrip-480p": "WEBRip-480p",
  "hdtv-720p": "HDTV-720p",
  "webrip-720p": "WEBRip-720p",
  "webdl-720p": "WEBDL-720p",
  "bluray-720p": "Bluray-720p",
  "hdtv-1080p": "HDTV-1080p",
  "webrip-1080p": "WEBRip-1080p",
  "webdl-1080p": "WEBDL-1080p",
  "bluray-1080p": "Bluray-1080p",
  "remux-1080p": "Remux-1080p",
  "hdtv-2160p": "HDTV-2160p",
  "webrip-2160p": "WEBRip-2160p",
  "webdl-2160p": "WEBDL-2160p",
  "bluray-2160p": "Bluray-2160p",
  "remux-2160p": "Remux-2160p",
  cam: "CAM",
};

const name = (quality: QualityKey) => names[quality];

test.each(golden.targets)("judgeMove agrees with Judge on every move of $name", (target) => {
  const rules = rulesOf(target);
  const disagreeing = target.moves.filter((move) => {
    const judged = judgeMove(rules, move);
    return (
      judged.allowed !== move.grabbable ||
      judged.rule !== move.rule ||
      reasonsOf(judged).join() !== move.reasons.join()
    );
  });
  expect(disagreeing).toEqual([]);
});

test("every sample move is Judge's answer for that move", () => {
  const everyMove = golden.targets.filter(
    ({ moves }) => moves.length === table.length * (table.length + 1),
  );
  expect(everyMove).toHaveLength(4);
  for (const target of everyMove) {
    for (const sample of upgradeSentence(rulesOf(target), table, name).samples) {
      const judged = target.moves.find(
        (move) => move.from?.quality === sample.from && move.to.quality === sample.to,
      );
      expect(judged?.grabbable).toBe(sample.allowed);
    }
  }
});

const plain = (quality: QualityKey, more: Partial<Side> = {}): Side => ({
  quality,
  version: 1,
  real: 0,
  repack: false,
  group: null,
  score: 0,
  ...more,
});

test("a repack, a minimum and the scores say why in words", () => {
  const scored: Rules = {
    tiers: [
      ["bluray-1080p", "remux-1080p"],
      ["webdl-1080p", "webrip-1080p"],
    ],
    cutoff: 0,
    scores: { minimum: 20, upgradeUntil: 100, increment: 10 },
    propers: "prefer_and_upgrade",
  };
  const said = (from: Side, to: Side) => {
    const { allowed, reason } = judgeInWords(scored, { from, to }, name);
    return [allowed, reason];
  };
  expect(
    said(
      plain("webdl-1080p", { group: "FLUX", score: 30 }),
      plain("webdl-1080p", { version: 2, repack: true, group: "NTb", score: 30 }),
    ),
  ).toEqual([false, "It's a repack from NTb, and the WEBDL-1080p file is from FLUX."]);
  expect(
    said(
      plain("webdl-1080p", { group: "FLUX", score: 30 }),
      plain("webdl-1080p", { version: 2, group: "NTb", score: 30 }),
    ),
  ).toEqual([true, "This release is a newer revision of the WEBDL-1080p file."]);
  expect(said(plain("webdl-1080p"), plain("bluray-1080p", { score: 10 }))).toEqual([
    false,
    "Bluray-1080p ranks above WEBDL-1080p. Score 10 is below this Target's minimum of 20.",
  ]);
  expect(said(plain("remux-1080p", { score: 30 }), plain("bluray-1080p", { score: 35 }))).toEqual([
    false,
    "Bluray-1080p scores 35. It needs 40 or more to replace Remux-1080p.",
  ]);
  expect(said(plain("remux-1080p", { score: 30 }), plain("bluray-1080p", { score: 40 }))).toEqual([
    true,
    "Bluray-1080p scores 40 and Remux-1080p scores 30, a gain of at least 10.",
  ]);
  expect(said(plain("webdl-1080p", { score: 100 }), plain("webrip-1080p", { score: 200 }))).toEqual(
    [
      false,
      "WEBDL-1080p and WEBRip-1080p are tied, and WEBDL-1080p scores 100, at or above this Target's upgrade-until score of 100.",
    ],
  );
  expect(said(plain("webdl-1080p", { version: 2 }), plain("webdl-1080p", { score: 50 }))).toEqual([
    false,
    "This release is an older revision of the WEBDL-1080p file.",
  ]);
});

test("the starter Target grabs any 720p and upgrades to the 1080p discs", () => {
  const starter = golden.targets.find((target) => target.name === "1080p");
  if (starter === undefined) {
    throw new Error("the golden has no 1080p Target");
  }
  const { sentence, samples } = upgradeSentence(rulesOf(starter), table, name);
  expect(sentence).toBe(
    "Grabs HDTV-720p, WEBRip-720p or WEBDL-720p or better. Upgrades until Bluray-1080p or Remux-1080p, then stops.",
  );
  expect(samples.map(({ from, to, allowed, reason }) => [from, to, allowed, reason])).toEqual([
    ["hdtv-720p", "bluray-720p", true, "Bluray-720p ranks above HDTV-720p."],
    ["hdtv-720p", "bluray-1080p", true, "Bluray-1080p ranks above HDTV-720p."],
    ["bluray-1080p", "remux-1080p", false, "Bluray-1080p and Remux-1080p are tied."],
    ["hdtv-1080p", "webrip-1080p", false, "HDTV-1080p and WEBRip-1080p are tied."],
    ["hdtv-1080p", "bluray-720p", false, "Bluray-720p ranks below HDTV-1080p."],
    ["hdtv-720p", "remux-2160p", false, "This Target doesn't accept Remux-2160p."],
  ]);
});

test("a cutoff at the bottom tier keeps the first file", () => {
  const { sentence, samples } = upgradeSentence(
    unscored({ tiers: [["bluray-1080p"], ["webdl-1080p", "webrip-1080p"]], cutoff: 1 }),
    table,
    name,
  );
  expect(sentence).toBe(
    "Grabs WEBDL-1080p or WEBRip-1080p or better. Keeps the first file it grabs.",
  );
  expect(samples[0]).toMatchObject({
    from: "webdl-1080p",
    to: "bluray-1080p",
    allowed: false,
    reason: "WEBDL-1080p has reached the cutoff, so no better tier replaces it.",
  });
});

test("a cutoff below the top lets files above it in but stops upgrading at it", () => {
  const { sentence, samples } = upgradeSentence(
    unscored({ tiers: [["remux-1080p"], ["bluray-1080p"], ["webdl-1080p"]], cutoff: 1 }),
    table,
    name,
  );
  expect(sentence).toBe("Grabs WEBDL-1080p or better. Upgrades until Bluray-1080p, then stops.");
  expect(samples.map(({ from, to, allowed }) => [from, to, allowed])).toEqual([
    ["webdl-1080p", "bluray-1080p", true],
    ["webdl-1080p", "remux-1080p", true],
    ["bluray-1080p", "remux-1080p", false],
    ["webdl-1080p", "remux-2160p", false],
  ]);
});

test("one tier grabs and never upgrades, and no tier grabs nothing", () => {
  expect(
    upgradeSentence(unscored({ tiers: [["webdl-1080p"]], cutoff: 0 }), table, name).sentence,
  ).toBe("Grabs WEBDL-1080p. Keeps the first file it grabs.");
  expect(upgradeSentence(unscored({ tiers: [], cutoff: 0 }), table, name)).toEqual({
    sentence: "Accepts no quality yet, so it grabs nothing.",
    samples: [],
  });
});

test("the filter pickers offer Go's vocabularies", () => {
  const values = z.record(z.string(), z.array(z.string())).parse(filtersGolden);
  expect(
    Object.fromEntries(
      attributes
        .filter((attribute) => attribute.key !== "language")
        .map((attribute) => [attribute.key, attribute.values.map((value) => value.key)]),
    ),
  ).toEqual(values);
});
