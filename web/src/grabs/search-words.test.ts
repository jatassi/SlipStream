import { describe, expect, test } from "vitest";

import {
  GrabsBehind,
  GrabsGrab,
  GrabsGroupState,
  GrabsSearch,
} from "../api/gen/zod/slipStream.schemas.zod";
import type { Clock } from "../format";
import {
  aBehind,
  aGroupState,
  aSearchQueued,
  aStopped,
  aWaiting,
  aWorkSearching,
  queuedBehind,
  queuedForLimit,
  queuedGroupOf,
  queuedNow,
  searchingNow,
  searchingSeveral,
  searchingStarved,
  starvedByRSS,
} from "../test/searching";
import {
  behindWords,
  foundByWords,
  groupWords,
  lastWords,
  leadWords,
  notFoundWords,
  outcomeWords,
  queuedToast,
  stoppedWords,
  triggerWords,
  waitWords,
} from "./search-words";

const clock: Clock = { now: new Date("2026-11-14T12:00:00Z"), zone: "UTC" };

type Listing = { readonly options: readonly (string | { readonly value: unknown })[] };

/** The values a wire enum or union of literals lists, its null left out. */
const valuesOf = (schema: Listing | { unwrap: () => Listing }): string[] =>
  ("unwrap" in schema ? schema.unwrap() : schema).options
    .map((option) => (typeof option === "string" ? option : option.value))
    .filter((value): value is string => typeof value === "string");

describe("every wire value has words", () => {
  test.each([
    ["run trigger", GrabsSearch.shape.trigger, triggerWords],
    ["run outcome", GrabsSearch.shape.outcome, outcomeWords],
    ["group state", GrabsGroupState.shape.state, groupWords],
    ["last result", GrabsGroupState.shape.lastResult, lastWords],
    ["found by", GrabsGrab.shape.foundBy, foundByWords],
    ["queued from", GrabsBehind.shape.aheadFrom, triggerWords],
  ] as const)("%s", (_, schema, words) => {
    const values = valuesOf(schema);
    expect(values.length).toBeGreaterThan(0);
    expect(values.filter((value) => !(value in words))).toEqual([]);
  });
});

test("a run says what started it and how it finished", () => {
  expect(triggerWords.failed_grab).toBe("After a failed Grab");
  expect(outcomeWords.grabbed({ grabs: 3, candidates: 14 })).toBe("Grabbed 3 Releases");
  expect(outcomeWords.grabbed({ grabs: 1, candidates: 1 })).toBe("Grabbed 1 Release");
  expect(outcomeWords.nothing_acceptable({ grabs: 0, candidates: 14 })).toBe(
    "Found 14 Releases, but none fits a Target",
  );
  expect(foundByWords.rss({ rssIndexerName: "TorrentLeech" })).toBe("RSS sync from TorrentLeech");
});

test("a queued group's popover says what holds it", () => {
  expect(behindWords(aBehind(), clock)).toBe("Queued behind 140 Works from Search all.");
  expect(behindWords(queuedForLimit(1).behind, clock)).toBe(
    "Waiting for TorrentLeech's search limit until 14:00.",
  );
  expect(behindWords(starvedByRSS(1).behind, clock)).toBe(
    "Backlog search waits for TorrentLeech. RSS sync uses all of its search limit.",
  );
  expect(behindWords(null, clock)).toBeUndefined();
});

test("the hero leads with the searching group, counting the Work's groups in flight", () => {
  expect(leadWords(searchingSeveral, clock, true)).toEqual({
    kind: "searching",
    words: "Searching volume 3 of 10",
    behind: undefined,
  });
  expect(leadWords(searchingSeveral, clock, false)).toEqual({
    kind: "searching",
    words: "Searching",
    behind: undefined,
  });
});

test("a searching group whose line waits for a starved Indexer says what it waits for", () => {
  const searching = aWorkSearching({
    groups: [searchingStarved(1)],
    lead: { groupKey: searchingStarved(1).key, of: 1, wait: null },
  });
  expect(leadWords(searching, clock, false)).toEqual({
    kind: "searching",
    words: "Searching",
    behind: "Backlog search waits for TorrentLeech. RSS sync uses all of its search limit.",
  });
});

test("the hero reads Queued with what holds it, then a wait, then the latest result", () => {
  const queued = aWorkSearching({
    groups: [queuedBehind(1)],
    lead: { groupKey: queuedBehind(1).key, of: 1, wait: null },
  });
  expect(leadWords(queued, clock, true)).toEqual({
    kind: "queued",
    words: "Queued",
    behind: "Queued behind 140 Works from Search all.",
  });
  const waiting = aWorkSearching({
    waits: [aWaiting(2)],
    lead: { groupKey: null, of: 0, wait: aWaiting(2) },
  });
  expect(leadWords(waiting, clock, true)?.words).toBe("Waiting for a better Release until 18:30");
  const found = aWorkSearching({
    groups: [
      aGroupState(1, { lastSearchedAt: "2026-11-13T09:12:00Z" }),
      aGroupState(2, {
        lastSearchedAt: "2026-11-14T09:12:00Z",
        nextSearchAt: "2026-11-15T14:00:00Z",
      }),
    ],
  });
  expect(leadWords(found, clock, true)).toEqual({
    kind: "result",
    result: "nothing_found",
    words: "Found nothing Today at 09:12. Next search Tomorrow at 14:00.",
  });
});

test("the hero shows nothing before the first search", () => {
  const never = aWorkSearching({
    groups: [aGroupState(1, { lastSearchedAt: null, lastResult: null })],
  });
  expect(leadWords(never, clock, true)).toBeUndefined();
});

test("the last result reads as the spec words it", () => {
  const at = { lastSearchedAt: "2026-11-14T09:12:00Z", nextSearchAt: "2026-11-15T14:00:00Z" };
  expect(lastWords.nothing_acceptable(aGroupState(1, { ...at, lastFound: 14 }), clock)).toBe(
    "Found 14 Releases, but none fits a Target. Next search Tomorrow at 14:00.",
  );
  expect(lastWords.grabbed(aGroupState(1, { lastGrabs: 3 }), clock)).toBe("Grabbed 3 Releases");
  expect(lastWords.no_indexer(aGroupState(1), clock)).toBe(
    "No Indexer could search. Check Indexers in Health.",
  );
});

describe("the toast after Search", () => {
  const pressed = aSearchQueued([queuedGroupOf(3)]);
  test("names what it searches when nothing holds the group", () => {
    const now = aWorkSearching({ groups: [queuedNow(3)] });
    expect(queuedToast(pressed, now, "12 episodes")).toBe("Searching 12 episodes");
    expect(queuedToast(pressed, aWorkSearching({ groups: [searchingNow(3)] }), "Dune")).toBe(
      "Searching Dune",
    );
  });
  test("says Queued while every group it queued waits behind something", () => {
    const now = aWorkSearching({ groups: [{ ...queuedBehind(3) }] });
    expect(queuedToast(pressed, now, "12 episodes")).toBe("Queued 12 episodes for search");
  });
  test("says Already searching when every group merged", () => {
    const merged = aSearchQueued([queuedGroupOf(3, "merged")]);
    expect(queuedToast(merged, aWorkSearching(), "12 episodes")).toBe("Already searching volume 3");
  });
});

test("a waiting cell says how many Releases it holds and when it grabs", () => {
  expect(waitWords(2, "2026-11-14T22:40:00Z", clock)).toBe(
    "2 Releases found so far. SlipStream grabs the best one at 22:40 unless a better one arrives.",
  );
  expect(waitWords(1, "2026-11-15T09:00:00Z", clock)).toBe(
    "1 Release found so far. SlipStream grabs the best one Tomorrow at 09:00 unless a better one arrives.",
  );
});

test("a stopped cell counts its failed Grabs and names Backlog search's next time", () => {
  expect(stoppedWords(aStopped(1), clock)).toBe(
    "3 Grabs failed here in 2 days. SlipStream searches again at Backlog pace, next Fri at 22:00.",
  );
  expect(stoppedWords(aStopped(1, { nextSearchAt: null }), clock)).toBe(
    "3 Grabs failed here in 2 days. Backlog search is off, so only Search again searches it.",
  );
});

test("a Not found row says when the last search found nothing and when SlipStream searches again", () => {
  expect(notFoundWords("2026-11-09T09:12:00Z", "2026-11-15T14:00:00Z", clock)).toBe(
    "Found nothing on Mon at 09:12. SlipStream searches again Tomorrow at 14:00.",
  );
  expect(notFoundWords("2026-11-14T09:12:00Z", null, clock)).toBe("Found nothing Today at 09:12.");
});
