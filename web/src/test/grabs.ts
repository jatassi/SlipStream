import type { GrabsGrab } from "../api/gen/model";
import { orchardId, orchardItemId } from "./library";
import { seedboxId } from "./nodes";

export const grabId = "6c8e0a2b-4d6f-4b8a-9c1e-5f7a9b1d3e5f";
export const indexerId = "0e2a4c6e-8f1b-4d3f-a5c7-e9b1d3f5a7c9";

const targetId = "1f3e5a7c-9b2d-4f6e-8a0c-2e4f6a8c0b1d";

const fetched: GrabsGrab = {
  id: grabId,
  workId: orchardId,
  itemId: orchardItemId,
  items: [orchardItemId],
  edition: "standard",
  targetIds: [targetId],
  targets: [{ targetId, itemId: orchardItemId, forced: false, settled: null }],
  releaseTitle: "The.Quiet.Orchard.2021.1080p.WEB-DL",
  releaseWords: "WEBDL-1080p · 700 MB",
  quality: "webdl-1080p",
  releaseSize: 734_003_200,
  indexerId,
  trackerId: null,
  nodeId: "4a6c8e0f-2b4d-4f6a-8c0e-2a4c6e8f0b1d",
  fetchNodeId: null,
  torrentId: "3e5a7c9b-1d3f-4a5c-8e7a-9c1b3d5f7a9c",
  downloadClient: null,
  infoHash: "0102030405060708090a0b0c0d0e0f1011121314",
  stage: "fetching",
  inFlight: true,
  history: [{ stage: "fetching", at: "2026-10-12T09:01:00Z" }],
  createdAt: "2026-10-12T09:01:00Z",
  source: "search",
  grabbedBy: null,
  manualSearchId: null,
  overrides: [],
  actions: { cancel: false, blocklistable: false, sendTo: false },
};

export const waitingToFetch: GrabsGrab = {
  ...fetched,
  torrentId: null,
  infoHash: null,
  fetchWait: {
    sentence: "Waiting to fetch the .torrent from Fixture Site. SlipStream retries at 14:20.",
    until: "2026-10-12T14:20:00Z",
  },
};

/**
 * A private Tracker's Grab whose route is the built-in client on seedbox,
 * waiting to fetch while seedbox is offline.
 */
export const waitingForSeedbox: GrabsGrab = {
  ...fetched,
  nodeId: seedboxId,
  torrentId: null,
  infoHash: null,
  waitsFor: { nodeId: seedboxId, client: null, off: false },
};

export const downloading: GrabsGrab = {
  ...fetched,
  fetchNodeId: fetched.nodeId,
  stage: "downloading",
  history: [...fetched.history, { stage: "downloading", at: "2026-10-12T09:01:02Z" }],
  progress: { done: 183_500_800, total: 734_003_200 },
};

export const imported: GrabsGrab = {
  ...fetched,
  stage: "imported",
  inFlight: false,
  targets: [{ targetId, itemId: orchardItemId, forced: false, settled: "placed" }],
  history: [
    ...downloading.history,
    { stage: "probing", at: "2026-10-12T09:03:00Z" },
    { stage: "importing", at: "2026-10-12T09:03:01Z" },
    { stage: "imported", at: "2026-10-12T09:03:02Z" },
  ],
};

/** An import that failed on a permission error, which needs an Admin. */
export const failedImport: GrabsGrab = {
  ...imported,
  stage: "failed",
  targets: fetched.targets,
  history: [...imported.history.slice(0, -1), { stage: "failed", at: "2026-10-12T09:03:02Z" }],
  failure: {
    stage: "importing",
    code: "import_failed",
    message: "place movies/The Quiet Orchard (2021): permission denied",
    needsYou: true,
    current: true,
  },
};
