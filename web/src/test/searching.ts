import { http, HttpResponse } from "msw";

import type {
  GrabsBehind,
  GrabsCellKey,
  GrabsGrabbing,
  GrabsGroupState,
  GrabsLead,
  GrabsQueuedGroup,
  GrabsSearchQueued,
  GrabsStopped,
  GrabsWaiting,
  GrabsWorkSearching,
  WorksLevel,
} from "../api/gen/model";
import { longShelf } from "./anthology";
import { torrentLeech } from "./indexers";
import { starter } from "./targets";

const volumes: readonly WorksLevel[] = longShelf.levels.filter((l) => !l.aside);
function volume(n: number): WorksLevel {
  const level = volumes[n - 1];
  if (level === undefined) throw new Error(`The Long Shelf has no Volume ${n}`);
  return level;
}

/** A search group's key as the server writes it for a level. */
export const groupKey = (level: WorksLevel) => `l:${level.id.replaceAll("-", "")}`;

/** A Volume's search group, idle and last searched with nothing found. */
export const aGroupState = (n: number, over: Partial<GrabsGroupState> = {}): GrabsGroupState => ({
  key: groupKey(volume(n)),
  label: volume(n).label.toLowerCase(),
  state: "idle",
  lane: null,
  trigger: null,
  again: false,
  behind: null,
  lastSearchedAt: "2026-11-09T09:12:00Z",
  lastResult: "nothing_found",
  lastFound: 0,
  lastGrabs: 0,
  nextSearchAt: "2026-11-16T22:00:00Z",
  ...over,
});

/** Queued by a press, first in the now lane. */
export const queuedNow = (n: number) =>
  aGroupState(n, { state: "queued", lane: "now", trigger: "search", nextSearchAt: null });

/** What holds a queued group: Works ahead of it in its lane. */
export const aBehind = (over: Partial<GrabsBehind> = {}): GrabsBehind => ({
  ahead: 140,
  aheadLane: "bulk",
  aheadFrom: "search_all",
  indexerId: null,
  indexerName: null,
  until: null,
  starved: false,
  ...over,
});

/** Queued behind 140 Works from Search all. */
export const queuedBehind = (n: number) =>
  aGroupState(n, {
    state: "queued",
    lane: "bulk",
    trigger: "search_all",
    behind: aBehind(),
    nextSearchAt: null,
  });

/** Waiting for TorrentLeech's search limit until 14:00. */
export const queuedForLimit = (n: number) =>
  aGroupState(n, {
    state: "queued",
    lane: "backlog",
    trigger: "backlog",
    behind: aBehind({
      ahead: 0,
      aheadLane: null,
      aheadFrom: null,
      indexerId: torrentLeech.id,
      indexerName: torrentLeech.name,
      until: "2026-11-14T14:00:00Z",
    }),
    nextSearchAt: null,
  });

/** Backlog search waits for TorrentLeech, whose search limit RSS sync uses up. */
export const starvedByRSS = (n: number) =>
  aGroupState(n, {
    state: "queued",
    lane: "backlog",
    trigger: "backlog",
    behind: aBehind({
      ahead: 0,
      aheadLane: null,
      aheadFrom: null,
      indexerId: torrentLeech.id,
      indexerName: torrentLeech.name,
      until: "2026-11-14T15:00:00Z",
      starved: true,
    }),
    nextSearchAt: null,
  });

/** Searching now for a press. */
export const searchingNow = (n: number) =>
  aGroupState(n, { state: "searching", lane: "now", trigger: "search", nextSearchAt: null });

/** Searching, with the same group queued again behind it. */
export const searchingAgain = (n: number) => ({ ...searchingNow(n), again: true });

/** Never searched, while Backlog search is off. */
export const neverSearched = (n: number) =>
  aGroupState(n, { lastSearchedAt: null, lastResult: null, nextSearchAt: null });

/** A cell of a Volume's first Story at the starter Target. */
export const aCell = (n: number): GrabsCellKey => {
  const item = longShelf.items.find((it) => it.levelId === volume(n).id);
  return { itemId: item?.id ?? "", edition: "any", targetId: starter.id };
};

/** A cell holding two Releases until 18:30 for a better one. */
export const aWaiting = (n: number, over: Partial<GrabsWaiting> = {}): GrabsWaiting => ({
  cell: aCell(n),
  until: "2026-11-14T18:30:00Z",
  candidates: 2,
  ...over,
});

/** A cell SlipStream stopped searching again for after three failed Grabs. */
export const aStopped = (n: number, over: Partial<GrabsStopped> = {}): GrabsStopped => ({
  cell: aCell(n),
  failures: [
    {
      code: "download_failed",
      at: "2026-11-12T08:00:00Z",
      grabId: "0199f0b2-0003-7000-8000-000000000001",
    },
    {
      code: "no_video_file",
      at: "2026-11-12T20:00:00Z",
      grabId: "0199f0b2-0003-7000-8000-000000000002",
    },
    { code: "download_failed", at: "2026-11-13T09:00:00Z", grabId: null },
  ],
  nextSearchAt: "2026-11-20T22:00:00Z",
  ...over,
});

/** The hero's line about one group of several queued or searching. */
export const aLead = (over: Partial<GrabsLead> = {}): GrabsLead => ({
  groupKey: groupKey(volume(3)),
  of: 10,
  wait: null,
  ...over,
});

/** The hero's line for a Work with nothing queued or searching. */
export const idleLead: GrabsLead = { groupKey: null, of: 0, wait: null };

/** A Work's search state: every Volume idle unless over names it. */
export const aWorkSearching = (over: Partial<GrabsWorkSearching> = {}): GrabsWorkSearching => ({
  groups: volumes.map((_, i) => aGroupState(i + 1)),
  waits: [],
  stops: [],
  lead: idleLead,
  ...over,
});

/** Volume 3 searching, Volumes 4 to 12 queued behind it. */
export const searchingSeveral = aWorkSearching({
  groups: volumes.map((_, i) => {
    const n = i + 1;
    if (n === 3) return searchingNow(n);
    return n > 3 && n <= 12 ? queuedNow(n) : aGroupState(n);
  }),
  lead: aLead(),
});

/** Only waits: Volume 2's first Story holds two Releases until 18:30. */
export const onlyWaiting = aWorkSearching({
  waits: [aWaiting(2)],
  lead: { groupKey: null, of: 0, wait: aWaiting(2) },
});

/** searchWork's answer for groups queued, or merged into one already queued. */
export const aSearchQueued = (groups: readonly GrabsQueuedGroup[]): GrabsSearchQueued => ({
  groups: [...groups],
});

export const queuedGroupOf = (
  n: number,
  state: GrabsQueuedGroup["state"] = "queued",
): GrabsQueuedGroup => ({
  key: groupKey(volume(n)),
  label: volume(n).label.toLowerCase(),
  itemIds: longShelf.items.filter((it) => it.levelId === volume(n).id).map((it) => it.id),
  state,
});

/** Serves a Work's search state, which a test changes, and records each Grab now. */
export function searchingServer(initial: GrabsWorkSearching = aWorkSearching()) {
  let state = initial;
  const grabbed: unknown[] = [];
  return {
    grabbed,
    set(next: GrabsWorkSearching) {
      state = next;
    },
    handlers: [
      http.get("*/api/v1/works/:workId/searching", () => HttpResponse.json(state)),
      http.post("*/api/v1/works/:workId/waits/grab", async ({ request }) => {
        grabbed.push(await request.json());
        const answer: GrabsGrabbing = { grabbing: state.waits.length };
        state = { ...state, waits: [], lead: idleLead };
        return HttpResponse.json(answer, { status: 202 });
      }),
    ],
  };
}

/** The Work's search queue a test moves, answering each press of Search with pressed. */
export function queueWorld({
  workId,
  initial,
  pressed,
  send,
}: {
  workId: string;
  initial: GrabsWorkSearching;
  pressed: GrabsSearchQueued;
  send: (event: { event: "grabs.queue-changed"; data: { workIds: string[] } }) => void;
}) {
  const queue = searchingServer(initial);
  const searched: string[] = [];
  return {
    searched,
    grabbed: queue.grabbed,
    handlers: [
      ...queue.handlers,
      http.post("*/api/v1/works/:workId/search", ({ params }) => {
        searched.push(String(params["workId"]));
        return HttpResponse.json(pressed, { status: 202 });
      }),
    ],
    moves: (next: GrabsWorkSearching) => {
      queue.set(next);
      send({ event: "grabs.queue-changed", data: { workIds: [workId] } });
    },
  };
}
