import { expect, test } from "vitest";

import type { TrackersWindow } from "../api/gen/model";
import { asEnded, endedLine, owesWords, windowSentences } from "./window-words";

const clock = { now: new Date("2026-10-12T12:00:00Z"), zone: "UTC" };

const tenDays = { clears: "time", ratio: null, seedSeconds: 864_000 } as const;

const owing: TrackersWindow = {
  infoHash: "a".repeat(40),
  name: "Hold.Fast.2024.1080p.WEB-DL",
  state: "owing",
  torrents: [],
  rule: { clears: "either", ratio: 1, seedSeconds: 864_000 },
  openedAt: "2026-10-01T09:00:00Z",
  metAt: null,
  closedAt: null,
  regrabbedAt: null,
  owed: null,
  owes: { seconds: 273_600, bytes: 1_288_490_189 },
  atRisk: "no",
  quietSince: null,
};

test("an open window says what it still owes, joined as its rule clears", () => {
  expect(owesWords(owing)).toBe("Owes 3 days 4 h of seeding or 1.2 GB of upload");
  expect(owesWords({ ...owing, rule: { ...tenDays, clears: "both" } })).toBe(
    "Owes 3 days 4 h of seeding and 1.2 GB of upload",
  );
  expect(owesWords({ ...owing, rule: null, owes: null })).toBe("No rule known");
});

test("an ended window says when it was met, missed or released", () => {
  expect(endedLine({ ...owing, state: "met", metAt: "2026-10-12T09:30:00Z" }, clock)).toBe(
    "Met at 09:30",
  );
  expect(endedLine({ ...owing, state: "missed", closedAt: "2026-10-03T18:05:00Z" }, clock)).toMatch(
    /^Missed at Oct 3, 18:05$/u,
  );
  expect(endedLine({ ...owing, state: "released", closedAt: null }, clock)).toBe("Released");
});

test("a window's sentences say how it stands at risk, after a regrab, and what a miss still owed", () => {
  expect(windowSentences(owing, "seedpool", clock)).toEqual([]);
  expect(
    windowSentences(
      { ...owing, atRisk: "yes", quietSince: "2026-10-12T08:00:00Z" },
      "seedpool",
      clock,
    ),
  ).toEqual(["It stopped seeding at 08:00. Seed it again before seedpool counts a hit and run."]);
  expect(windowSentences({ ...owing, atRisk: "unknown" }, "seedpool", clock)).toEqual([
    "Every Node that holds it is offline, so whether it's seeding isn't known.",
  ]);
  expect(
    windowSentences(
      { ...owing, closedAt: "2026-10-12T08:00:00Z", regrabbedAt: "2026-10-12T10:00:00Z" },
      "seedpool",
      clock,
    ),
  ).toEqual(["It was missed at 08:00 and grabbed again at 10:00."]);
  expect(
    windowSentences(
      {
        ...owing,
        state: "missed",
        rule: tenDays,
        closedAt: "2026-10-12T08:00:00Z",
        owed: { seconds: 86_400, bytes: 0 },
        atRisk: "yes",
      },
      "seedpool",
      clock,
    ),
  ).toEqual(["It still owed 1 day of seeding when its last torrent was removed."]);
});

test("the history tells a window grabbed again after its miss as that miss, and drops a regrab older than its last miss", () => {
  const regrabbed = {
    ...owing,
    rule: tenDays,
    closedAt: "2026-10-12T08:00:00Z",
    regrabbedAt: "2026-10-12T10:00:00Z",
    owed: { seconds: 86_400, bytes: 0 },
  };
  expect(endedLine(asEnded(regrabbed), clock)).toBe("Missed at 08:00");
  expect(windowSentences(asEnded(regrabbed), "seedpool", clock)).toEqual([
    "It still owed 1 day of seeding when its last torrent was removed.",
    "It was grabbed again at 10:00.",
  ]);
  expect(
    windowSentences(
      { ...regrabbed, state: "missed", closedAt: "2026-10-12T11:00:00Z" },
      "seedpool",
      clock,
    ),
  ).toEqual(["It still owed 1 day of seeding when its last torrent was removed."]);
});
