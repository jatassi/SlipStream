import { expect, test } from "vitest";

import { goalSentence, noRuleSentence, ruleSentence } from "./words";

const day = 86_400;

const rule = (
  clears: "time" | "ratio" | "either" | "both",
  ratio: number | null,
  days: number | null,
) =>
  ruleSentence("TorrentLeech", {
    clears,
    ratio,
    seedSeconds: days === null ? null : days * day,
    source: "releases",
  });

test("a rule reads as what the Tracker requires for each torrent, each way it clears", () => {
  expect(rule("time", null, 10)).toBe(
    "TorrentLeech requires 10 days of seeding for each torrent. Ratio doesn't count.",
  );
  expect(rule("ratio", 1, null)).toBe(
    "TorrentLeech requires a ratio of 1.0 for each torrent. Seeding time doesn't count.",
  );
  expect(rule("either", 1, 10)).toBe(
    "TorrentLeech requires a ratio of 1.0 or 10 days of seeding for each torrent.",
  );
  expect(rule("both", 1.25, 1)).toBe(
    "TorrentLeech requires a ratio of 1.25 and 1 day of seeding for each torrent.",
  );
});

test("a Tracker with no rule says why", () => {
  expect(noRuleSentence("nyaa", "public")).toBe(
    "nyaa is public and has no seeding rule. Its torrents seed until your seeding goal is met.",
  );
  expect(noRuleSentence("seedpool", "private")).toBe(
    "seedpool's releases don't include a seeding rule. SlipStream keeps seeding its private torrents until you set one.",
  );
});

test("a goal reads as how far each torrent seeds and what happens then", () => {
  expect(
    goalSentence({ combine: "first", ratio: 2, seedSeconds: 14 * day, action: "remove-delete" }),
  ).toBe(
    "Each torrent seeds until it reaches a ratio of 2.0 or 14 days of seeding, whichever comes first. Then SlipStream removes it and deletes its download copy.",
  );
  expect(goalSentence({ combine: "both", ratio: 1, seedSeconds: 3 * day, action: "pause" })).toBe(
    "Each torrent seeds until it reaches a ratio of 1.0 and 3 days of seeding. Then SlipStream pauses it.",
  );
  expect(goalSentence({ combine: "first", ratio: null, seedSeconds: 7200, action: "remove" })).toBe(
    "Each torrent seeds until it reaches 2 h of seeding. Then SlipStream removes it.",
  );
  expect(goalSentence({ combine: "forever", ratio: null, seedSeconds: null, action: null })).toBe(
    "Each torrent seeds until you remove it.",
  );
});
