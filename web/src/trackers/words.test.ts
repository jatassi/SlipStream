import { expect, test } from "vitest";

import { goalSentence, noRuleSentence, ruleSentence, standingSentence } from "./words";

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
    "nyaa is public, so its torrents follow your seeding goal alone.",
  );
  expect(noRuleSentence("seedpool", "private")).toBe(
    "seedpool's Releases don't state a seeding rule, so SlipStream never stops its private torrents. Set seedpool's rule to let its goal run.",
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
    "Each torrent seeds with no end until you remove it.",
  );
});

test("the standing sentence counts what is seeding, what still owes and what is at risk", () => {
  const counts = { torrents: 214, seeding: 214, owing: 0, atRisk: 0, goalMet: 0 };
  expect(standingSentence("seedpool", { ...counts, torrents: 0, seeding: 0 })).toBe(
    "SlipStream has no torrents from seedpool.",
  );
  expect(standingSentence("seedpool", counts)).toBe(
    "214 torrents are seeding. None of them owe seedpool seeding.",
  );
  expect(standingSentence("seedpool", { ...counts, owing: 1 })).toBe(
    "214 torrents are seeding. 1 still owes seedpool seeding.",
  );
  expect(standingSentence("seedpool", { ...counts, seeding: 1, owing: 3, atRisk: 2 })).toBe(
    "1 torrent is seeding. 3 still owe seedpool seeding, and 2 of those aren't seeding.",
  );
  expect(
    standingSentence("seedpool", { ...counts, torrents: 1, seeding: 0, owing: 1, atRisk: 1 }),
  ).toBe(
    "Its one torrent isn't seeding. 1 still owes seedpool seeding, and 1 of those isn't seeding.",
  );
});
