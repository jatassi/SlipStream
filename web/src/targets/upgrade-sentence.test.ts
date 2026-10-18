import { expect, test } from "vitest";
import { z } from "zod";

import filtersGolden from "../../../internal/targets/testdata/filters.json" with { type: "json" };
import upgradesGolden from "../../../internal/targets/testdata/upgrades.json" with { type: "json" };
import type { Layout, QualityKey } from "./order";
import { judgeMove, upgradeSentence } from "./upgrade-sentence";
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

const golden = z
  .object({
    targets: z.array(
      z.object({
        name: z.string(),
        tiers: z.array(z.array(key)),
        cutoff: key,
        moves: z.array(
          z.object({
            from: key.nullable(),
            to: key,
            grabbable: z.boolean(),
            reasons: z.array(z.enum(["quality_not_in_target", "not_an_upgrade"])),
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

const reasonsOf = {
  not_in_target: ["quality_not_in_target"],
  final: ["not_an_upgrade"],
  tied: ["not_an_upgrade"],
  lower: ["not_an_upgrade"],
  upgrade: [],
  vacant: [],
  outside: [],
} as const;

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

test.each(golden.targets)(
  "judgeMove agrees with Judge on every move of $name",
  ({ tiers, cutoff, moves }) => {
    const layout = layoutOf(tiers, cutoff);
    const disagreeing = moves.filter((move) => {
      const judged = judgeMove(layout, move);
      return (
        judged.allowed !== move.grabbable || reasonsOf[judged.why].join() !== move.reasons.join()
      );
    });
    expect(disagreeing).toEqual([]);
  },
);

test("every sample move is Judge's answer for that move", () => {
  for (const { tiers, cutoff, moves } of golden.targets) {
    for (const sample of upgradeSentence(layoutOf(tiers, cutoff), table, name).samples) {
      const judged = moves.find((move) => move.from === sample.from && move.to === sample.to);
      expect(judged?.grabbable).toBe(sample.allowed);
    }
  }
});

test("the starter Target grabs any 720p and upgrades to the 1080p discs", () => {
  const starter = golden.targets.find((target) => target.name === "1080p");
  if (starter === undefined) {
    throw new Error("the golden has no 1080p Target");
  }
  const { sentence, samples } = upgradeSentence(
    layoutOf(starter.tiers, starter.cutoff),
    table,
    name,
  );
  expect(sentence).toBe(
    "Grabs HDTV-720p, WEBRip-720p or WEBDL-720p or better. Upgrades until Bluray-1080p or Remux-1080p, then stops.",
  );
  expect(samples.map(({ from, to, allowed, reason }) => [from, to, allowed, reason])).toEqual([
    ["hdtv-720p", "bluray-720p", true, "Bluray-720p ranks above HDTV-720p."],
    ["hdtv-720p", "bluray-1080p", true, "Bluray-1080p ranks above HDTV-720p."],
    [
      "bluray-1080p",
      "remux-1080p",
      false,
      "Bluray-1080p has reached the cutoff, so nothing replaces it.",
    ],
    ["hdtv-1080p", "webrip-1080p", false, "HDTV-1080p and WEBRip-1080p are tied."],
    ["hdtv-1080p", "bluray-720p", false, "Bluray-720p ranks below HDTV-1080p."],
    ["hdtv-720p", "remux-2160p", false, "This Target doesn't accept Remux-2160p."],
  ]);
});

test("a cutoff at the bottom tier keeps the first file", () => {
  const { sentence, samples } = upgradeSentence(
    { tiers: [["bluray-1080p"], ["webdl-1080p", "webrip-1080p"]], cutoff: 1 },
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
    reason: "WEBDL-1080p has reached the cutoff, so nothing replaces it.",
  });
});

test("a cutoff below the top lets files above it in but stops upgrading at it", () => {
  const { sentence, samples } = upgradeSentence(
    { tiers: [["remux-1080p"], ["bluray-1080p"], ["webdl-1080p"]], cutoff: 1 },
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
  expect(upgradeSentence({ tiers: [["webdl-1080p"]], cutoff: 0 }, table, name).sentence).toBe(
    "Grabs WEBDL-1080p. Keeps the first file it grabs.",
  );
  expect(upgradeSentence({ tiers: [], cutoff: 0 }, table, name)).toEqual({
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
