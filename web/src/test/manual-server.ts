import { HttpResponse, http } from "msw";

import type {
  GrabsBlocklist,
  GrabsGrabPick,
  GrabsManualSearch,
  GrabsOffer,
  GrabsPickResult,
  GrabsStartManual,
} from "../api/gen/model";
import {
  GrabFromManualSearchBody,
  RemoveFromBlocklistBody,
  StartManualSearchBody,
} from "../api/gen/zod/grabs/grabs.zod";
import { blocklistPage } from "./blocklist";
import { blocked, flux, lowScore, lowScoreChoice, manualSearchId, release, runId } from "./manual";

/** All Releases for the film: FLUX ranked first, the low-score one with its lift, the blocked one, and one no Target takes. */
export const filmReleases: GrabsOffer[] = [
  {
    release: flux,
    rank: 1,
    score: 30,
    quality: 9,
    grab: {
      grabbable: true,
      outcome: "It gets you 1080p for The Quiet Orchard (2021).",
      cells: ["cell-orchard-1080p"],
      lifts: [],
      edition: null,
      cancels: [],
    },
  },
  {
    release: lowScore,
    rank: 2,
    score: 5,
    quality: 8,
    grab: {
      grabbable: true,
      outcome: "It gets you 1080p for The Quiet Orchard (2021).",
      cells: ["cell-orchard-1080p"],
      lifts: lowScoreChoice.lifts,
      edition: null,
      cancels: [],
    },
  },
  {
    release: blocked,
    rank: 3,
    score: 20,
    quality: 10,
    grab: {
      grabbable: true,
      outcome: "It gets you 1080p for The Quiet Orchard (2021).",
      cells: ["cell-orchard-1080p"],
      lifts: [],
      edition: null,
      cancels: [],
    },
  },
  {
    release: release(
      `${runId}.7`,
      "The.Quiet.Orchard.2021.720p.HDTV.x264-KILLERS",
      "HDTV-720p · KILLERS · 1.1 GB",
      { verdicts: ["1080p doesn't accept HDTV-720p.", "2160p HDR doesn't accept HDTV-720p."] },
    ),
    rank: null,
    score: 0,
    quality: 0,
    grab: {
      grabbable: false,
      cells: [],
      lifts: [],
      edition: null,
      cancels: [],
      refusal: "No Target here takes HDTV-720p.",
    },
  },
];

/**
 * The Manual search endpoints for one Work: startManualSearch answers 202,
 * getManualSearch answers `plan(pins)`, and a grab answers `results` per pick
 * or a Grab id for each. It keeps each start's scope, each GET's pins, each
 * grab's picks and each Blocklist removal's hashes.
 */
export function manualServer({
  plan,
  releases = filmReleases,
  blocklist = blocklistPage,
  results,
}: {
  plan: (pins: readonly string[]) => GrabsManualSearch;
  releases?: readonly GrabsOffer[];
  blocklist?: GrabsBlocklist;
  results?: (picks: readonly GrabsGrabPick[]) => GrabsPickResult[];
}) {
  const started: GrabsStartManual[] = [];
  const pinned: string[][] = [];
  const grabbed: GrabsGrabPick[][] = [];
  const unblocked: string[][] = [];
  const handlers = [
    http.post("*/api/v1/works/:workId/manual-search", async ({ request }) => {
      const text = await request.text();
      started.push(StartManualSearchBody.parse(text === "" ? {} : JSON.parse(text)));
      return HttpResponse.json({ manualSearchId }, { status: 202 });
    }),
    http.get("*/api/v1/works/:workId/manual-search", ({ request }) => {
      const pins = new URL(request.url).searchParams.get("pins")?.split(",") ?? [];
      pinned.push(pins);
      return HttpResponse.json(plan(pins));
    }),
    http.get("*/api/v1/works/:workId/manual-search/releases", () =>
      HttpResponse.json({ releases }),
    ),
    http.post("*/api/v1/works/:workId/manual-search/grabs", async ({ request }) => {
      const { picks } = GrabFromManualSearchBody.parse(await request.json());
      grabbed.push(picks);
      return HttpResponse.json({
        results:
          results?.(picks) ??
          picks.map((_, i) => ({
            grabId: `0199f3a1-5c2e-7d4f-8a6b-${String(i).padStart(12, "0")}`,
            problem: null,
          })),
      });
    }),
    http.get("*/api/v1/blocklist", () => HttpResponse.json(blocklist)),
    http.post("*/api/v1/blocklist/remove", async ({ request }) => {
      unblocked.push(RemoveFromBlocklistBody.parse(await request.json()).infoHashes);
      return new HttpResponse(null, { status: 204 });
    }),
  ];
  return { handlers, started, pinned, grabbed, unblocked };
}
