import { http, HttpResponse } from "msw";

import type {
  GrabsQueueStatus,
  GrabsQueuedCanceled,
  GrabsSearchQueued,
  GrabsSeveralQueued,
  GrabsWantedPage,
  GrabsWantedRow,
} from "../api/gen/model";
import { longShelf } from "./anthology";
import { torrentLeech } from "./indexers";
import { orchard } from "./library";

/** The queue's status line: 3 searching, 142 queued, Backlog search tonight at 22:00. */
export const aQueueStatus = (over: Partial<GrabsQueueStatus> = {}): GrabsQueueStatus => ({
  searching: 3,
  queued: 142,
  nextBacklogAt: "2026-11-14T22:00:00Z",
  limits: [],
  ...over,
});

/** The status line with TorrentLeech's search limit holding searches until 14:00. */
export const limitedStatus = aQueueStatus({
  limits: [
    {
      indexerId: torrentLeech.id,
      indexerName: torrentLeech.name,
      until: "2026-11-14T14:00:00Z",
      starved: false,
    },
  ],
});

/** The status line where RSS sync uses all of TorrentLeech's search limit. */
export const starvedStatus = aQueueStatus({
  limits: [
    {
      indexerId: torrentLeech.id,
      indexerName: torrentLeech.name,
      until: "2026-11-14T15:00:00Z",
      starved: true,
    },
  ],
});

/** The status line while Backlog search is off and nothing is queued. */
export const quietStatus = aQueueStatus({ searching: 0, queued: 0, nextBacklogAt: null });

/** The Quiet Orchard, missing, last searched Mon at 09:12 with nothing found. */
export const aWantedRow = (over: Partial<GrabsWantedRow> = {}): GrabsWantedRow => ({
  work: { id: orchard.id, contentType: orchard.contentType, label: orchard.label, posterUrl: null },
  reason: "missing",
  others: {},
  ranges: [],
  lastSearchedAt: "2026-11-09T09:12:00Z",
  lastResult: "nothing_found",
  nextSearchAt: "2026-11-16T22:00:00Z",
  state: "idle",
  ...over,
});

/** The Long Shelf, missing Stories V02S01 to V02S08, with 3 upgrades. */
export const missingRange = aWantedRow({
  work: {
    id: longShelf.id,
    contentType: longShelf.contentType,
    label: longShelf.label,
    posterUrl: null,
  },
  others: { upgrade: 3 },
  ranges: [{ first: "Story V02S01", last: "Story V02S08" }],
});

/** A Work whose last Grab failed, queued again. */
export const failedQueued = aWantedRow({
  reason: "failed",
  lastResult: "grabbed",
  state: "queued",
  nextSearchAt: null,
});

/** A Work searching now for an upgrade. */
export const upgradeSearching = aWantedRow({
  reason: "upgrade",
  state: "searching",
  nextSearchAt: null,
});

/** A Work whose only wanted files are gone, which only a press searches. */
export const goneOnly = aWantedRow({
  reason: "gone",
  lastSearchedAt: null,
  lastResult: null,
  nextSearchAt: null,
});

/** A Wanted page of rows under status, the last page unless next names one. */
export const aWantedPage = (
  rows: readonly GrabsWantedRow[],
  over: Partial<GrabsWantedPage> = {},
): GrabsWantedPage => ({
  status: aQueueStatus(),
  total: rows.length,
  leftGone: 0,
  rows: [...rows],
  next: null,
  ...over,
});

/** Serves Wanted's pages in order by cursor, Search on one Work or several and Cancel of the bulk lane. */
export function wantedServer(
  pages: readonly GrabsWantedPage[] = [aWantedPage([missingRange, failedQueued, upgradeSearching])],
  {
    several = { queued: 1, merged: 0, leftGone: 0, works: [] },
    one = { groups: [{ key: "w", label: "the Work", itemIds: ["i"], state: "queued" }] },
  }: { several?: GrabsSeveralQueued; one?: GrabsSearchQueued } = {},
) {
  const searched: unknown[] = [];
  const pressed: string[] = [];
  const asked: URLSearchParams[] = [];
  let canceled = 0;
  return {
    searched,
    pressed,
    asked,
    get canceled() {
      return canceled;
    },
    handlers: [
      http.get("*/api/v1/wanted", ({ request }) => {
        const params = new URL(request.url).searchParams;
        asked.push(params);
        const cursor = params.get("cursor");
        const page =
          cursor === null ? pages[0] : pages.find((_, i) => pages[i - 1]?.next === cursor);
        return HttpResponse.json(page ?? aWantedPage([]));
      }),
      http.post("*/api/v1/works/search", async ({ request }) => {
        searched.push(await request.json());
        return HttpResponse.json(several, { status: 202 });
      }),
      http.post("*/api/v1/works/:workId/search", ({ params }) => {
        pressed.push(String(params["workId"]));
        return HttpResponse.json(one, { status: 202 });
      }),
      http.delete("*/api/v1/search-queue", () => {
        canceled++;
        const answer: GrabsQueuedCanceled = { canceled: 140 };
        return HttpResponse.json(answer);
      }),
    ],
  };
}
