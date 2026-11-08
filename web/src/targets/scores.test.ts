import { expect, test } from "vitest";

import {
  type ScoresDraft,
  matching,
  scoresBody,
  scoresDraftOf,
  scoresInWords,
  scoresProblem,
  totalsOf,
  withScore,
} from "./scores";

const catalog = [
  { id: "f1", name: "BR-DISK" },
  { id: "f2", name: "DV HDR10+" },
  { id: "f3", name: "HD Bluray Tier 01" },
];

const loaded = scoresDraftOf(
  {
    formats: [
      { formatId: "f1", score: -10000 },
      { formatId: "f3", score: 1800 },
    ],
    minimum: 0,
    upgradeUntil: 1500,
    increment: 10,
  },
  catalog,
);

test("every Custom Format opens with the Target's score, 0 when it scores none", () => {
  expect(loaded.formats.map(({ name, score }) => [name, score])).toEqual([
    ["BR-DISK", "-10000"],
    ["DV HDR10+", "0"],
    ["HD Bluray Tier 01", "1800"],
  ]);
});

test("the body sends the scored formats and the three totals, leaving out formats at 0", () => {
  const edited: ScoresDraft = { ...withScore(loaded, "f2", " 25 "), minimum: "-5" };
  expect(scoresProblem(edited)).toBe(false);
  expect(scoresBody(withScore(edited, "f3", "0"))).toEqual({
    formats: [
      { formatId: "f1", score: -10000 },
      { formatId: "f2", score: 25 },
    ],
    minimum: -5,
    upgradeUntil: 1500,
    increment: 10,
  });
});

test.each([
  ["a score that isn't whole", withScore(loaded, "f2", "2.5")],
  ["an empty score", withScore(loaded, "f2", "")],
  ["an empty minimum", { ...loaded, minimum: "" }],
  ["an upgrade-until that isn't a number", { ...loaded, upgradeUntil: "lots" }],
  ["an increment of 0", { ...loaded, increment: "0" }],
] satisfies [string, ScoresDraft][])("%s stops the save", (_, scores) => {
  expect(scoresProblem(scores)).toBe(true);
});

test("a total mid-edit reads as its default", () => {
  expect(totalsOf({ ...loaded, minimum: "-", upgradeUntil: "", increment: "0" })).toEqual({
    minimum: 0,
    upgradeUntil: 0,
    increment: 1,
  });
});

test("the name filter ignores case and keeps the catalog's order", () => {
  expect(matching(loaded, " hd").map(({ name }) => name)).toEqual([
    "DV HDR10+",
    "HD Bluray Tier 01",
  ]);
  expect(matching(loaded, "remux")).toEqual([]);
  expect(matching(loaded, "")).toHaveLength(3);
});

test("the lead says how many formats score and what the three totals do", () => {
  expect(scoresInWords(loaded)).toEqual({
    lead: "Scores 2 of 3 Custom Formats. A Release needs a score of 0 or more to be grabbed. In the same tier, a Release replaces a file when it scores 10 more, until the file scores 1500.",
    warnings: [],
  });
});

test("a Target scoring nothing ranks by quality alone", () => {
  const unscored = scoresDraftOf(
    { formats: [], minimum: 0, upgradeUntil: 0, increment: 1 },
    catalog,
  );
  expect(scoresInWords(unscored).lead).toBe(
    "Scores none of the 3 Custom Formats, so this Target ranks Releases by quality alone. Score a Custom Format to prefer the Releases that match it.",
  );
  expect(scoresInWords({ ...unscored, formats: [] }).lead).toBe(
    "No Custom Formats exist yet, so this Target ranks Releases by quality alone.",
  );
});

test("a minimum or upgrade-until past the best possible score warns", () => {
  expect(scoresInWords({ ...loaded, minimum: "2000", upgradeUntil: "1801" }).warnings).toEqual([
    "No Release can reach the minimum of 2000, because the most a Release can score here is 1800. Lower the minimum or raise a score.",
    "No file can reach the upgrade-until score of 1801, because the most a Release can score here is 1800. Files stay wanted past the cutoff. Lower upgrade-until or raise a score.",
  ]);
  expect(scoresInWords({ ...loaded, minimum: "1800", upgradeUntil: "1800" }).warnings).toEqual([]);
});
