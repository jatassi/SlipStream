import { expect, test } from "vitest";

import type { Clock } from "../format";
import { aQueueStatus, aWantedRow, goneOnly, missingRange, starvedStatus } from "../test/wanted";
import {
  lastWords,
  nextWords,
  phoneLine,
  rangesWords,
  reasonWords,
  statusParts,
} from "./wanted-words";

const saturday: Clock = { now: new Date("2026-11-14T10:00:00Z"), zone: "UTC" };

test("a row's reason leads, with the count of each other reason after it", () => {
  expect(reasonWords(missingRange)).toBe("Missing. 3 upgrades.");
  expect(reasonWords(aWantedRow({ reason: "failed", others: { missing: 2, gone: 1 } }))).toBe(
    "Failed. 2 missing. 1 file gone.",
  );
});

test("Items read as runs in the Needs you form", () => {
  expect(
    rangesWords([
      { first: "S02E01", last: "S02E08" },
      { first: "S03E02", last: "S03E02" },
    ]),
  ).toBe("S02E01 to S02E08, S03E02");
});

test("the last search says how it went and when", () => {
  expect(lastWords(missingRange, saturday)).toBe("Found nothing on Mon at 09:12");
  expect(
    lastWords(
      aWantedRow({ lastResult: "nothing_found", lastSearchedAt: "2026-11-14T09:12:00Z" }),
      saturday,
    ),
  ).toBe("Found nothing Today at 09:12");
  expect(lastWords(goneOnly, saturday)).toBe("Not searched yet");
});

test("the next search names the state, the time, or that only a press searches", () => {
  expect(nextWords(aWantedRow({ nextSearchAt: "2026-11-15T14:00:00Z" }), saturday)).toBe(
    "Tomorrow at 14:00",
  );
  expect(nextWords(aWantedRow({ state: "queued" }), saturday)).toBe("Queued");
  expect(nextWords(aWantedRow({ state: "searching" }), saturday)).toBe("Searching");
  expect(nextWords(goneOnly, saturday)).toBe("Only when you search");
});

test("a phone row reads the reason, the Items and the next search as one line", () => {
  expect(
    phoneLine(
      {
        ...missingRange,
        ranges: [{ first: "S02E01", last: "S02E08" }],
        nextSearchAt: "2026-11-15T14:00:00Z",
      },
      saturday,
    ),
  ).toBe("Missing. S02E01 to S02E08. Next search Tomorrow at 14:00.");
});

test("the status sentence counts the queue, the next Backlog search and each limit", () => {
  expect(
    statusParts(aQueueStatus({ nextBacklogAt: "2026-11-14T22:00:00Z" }), saturday).map(
      ({ text }) => text,
    ),
  ).toEqual(["3 searching", "142 queued", "Next Backlog search Tonight at 22:00"]);
  expect(statusParts(starvedStatus, saturday).at(-1)).toEqual({
    kind: "warning",
    text: "RSS sync uses all of TorrentLeech's search limit",
    href: `/settings/indexers/${starvedStatus.limits[0]?.indexerId ?? ""}`,
  });
});
