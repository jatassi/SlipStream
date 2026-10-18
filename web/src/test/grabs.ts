import type { GrabsGrab } from "../api/gen/model";
import { orchardId, orchardItemId } from "./library";

export const grabId = "6c8e0a2b-4d6f-4b8a-9c1e-5f7a9b1d3e5f";

const fetched: GrabsGrab = {
  id: grabId,
  workId: orchardId,
  itemId: orchardItemId,
  edition: "standard",
  targetIds: ["1f3e5a7c-9b2d-4f6e-8a0c-2e4f6a8c0b1d"],
  releaseTitle: "The.Quiet.Orchard.2021.1080p.WEB-DL",
  releaseSize: 734_003_200,
  indexerId: "0e2a4c6e-8f1b-4d3f-a5c7-e9b1d3f5a7c9",
  nodeId: "4a6c8e0f-2b4d-4f6a-8c0e-2a4c6e8f0b1d",
  fetchNodeId: null,
  infoHash: "0102030405060708090a0b0c0d0e0f1011121314",
  stage: "fetching",
  history: [{ stage: "fetching", at: "2026-10-12T09:01:00Z" }],
  createdAt: "2026-10-12T09:01:00Z",
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
  history: [
    ...downloading.history,
    { stage: "probing", at: "2026-10-12T09:03:00Z" },
    { stage: "importing", at: "2026-10-12T09:03:01Z" },
    { stage: "imported", at: "2026-10-12T09:03:02Z" },
  ],
};
