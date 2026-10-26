import { HttpResponse, type JsonBodyType, http } from "msw";

import type {
  TransfersSettings,
  TransfersSpeed,
  TransfersTransfer,
  TransfersTransferState,
} from "../api/gen/model";
import { SetTransferSettingsBody } from "../api/gen/zod/transfers/transfers.zod";
import { homeId, seedboxId } from "./nodes";

const gib = 1024 * 1024 * 1024;
const quarter = Math.round(2.1 * gib);
const mib = 1024 * 1024;

/** A Transfer from seedbox into a root on home, copying 2.1 GB of 8.4 GB at 48 MB/s with 2 min left. */
export const copying: TransfersTransfer = {
  id: "7d3f1a9c-2b4e-4c6a-8e0f-1a3c5e7b9d2f",
  purpose: "grab",
  job: "3c8e1f5a-7b2d-4e9c-a1f3-5d7b9e2c4a6f",
  jobName: "Heat (1995)",
  source: { nodeId: seedboxId, nodeName: "seedbox" },
  destination: {
    kind: "root",
    rootId: "8f2a4c6e-1b3d-4f5a-9c7e-0d2f4a6b8c1e",
    nodeId: homeId,
    nodeName: "home",
  },
  state: "copying",
  verified: quarter,
  total: 4 * quarter,
  rate: 48 * mib,
  estimateSeconds: 110,
  route: "receiver_dials",
  files: [
    {
      position: 0,
      name: "Heat (1995).mkv",
      size: 4 * quarter - 40_000,
      verified: false,
      placed: false,
    },
    { position: 1, name: "Heat (1995).en.srt", size: 40_000, verified: true, placed: false },
  ],
  details: { lanes: 3, rttMs: 42, retried: 1 },
};

const at = (
  state: Exclude<TransfersTransferState, "copying">,
  id: string,
  jobName: string,
  reason: string,
): TransfersTransfer => {
  const { route: _, ...rest } = copying;
  return {
    ...rest,
    id,
    job: `${id}-job`,
    jobName,
    state,
    reason,
    verified: 0,
    rate: 0,
    estimateSeconds: 0,
    details: { lanes: 0, rttMs: 0, retried: 0 },
  };
};

export const queued = at(
  "queued",
  "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
  "Collateral (2004)",
  "Queued behind 1 other Transfer from seedbox.",
);

export const waiting: TransfersTransfer = {
  ...at(
    "waiting",
    "2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e",
    "Thief (1981)",
    "Waiting for seedbox to come back.",
  ),
  wait: { kind: "node_away", nodeId: seedboxId },
};

export const needsSpace = at(
  "needs_space",
  "3c4d5e6f-7a8b-4c9d-8e1f-2a3b4c5d6e7f",
  "Manhunter (1986)",
  "Waiting for 12 GB free on home (Media).",
);

export const failed = at(
  "failed",
  "4d5e6f7a-8b9c-4d0e-9f2a-3b4c5d6e7f8a",
  "Miami Vice (2006)",
  "Miami Vice (2006).mkv kept arriving damaged from seedbox. Check seedbox's memory.",
);

export const speedFromSeedbox: TransfersSpeed = {
  sourceNodeId: seedboxId,
  destinationNodeId: homeId,
  rate: 48 * mib,
  rateAt: "2026-10-26T14:02:00Z",
  route: "receiver_dials",
  routeAt: "2026-10-26T14:00:00Z",
};

export const sendLimitOf = (value: number): TransfersSettings => ({
  sendLimit: {
    kind: "rate",
    key: "transfers.sendLimit",
    label: "Send limit",
    source: "database",
    applies: "live",
    pendingRestart: false,
    isDefault: value === 0,
    value,
    running: value,
    default: 0,
    max: 2000 * mib,
    zero: "No limit",
  },
});

type Write = { nodeId: string; body: JsonBodyType };

/** The Transfers API over a list, the recent speeds and each Node's send limit, recording each settings write. */
export function transfersApi({
  transfers = [] as TransfersTransfer[],
  speeds = [] as TransfersSpeed[],
} = {}) {
  const state = { transfers, speeds, settings: sendLimitOf(0) };
  const writes: Write[] = [];
  const handlers = [
    http.get("*/api/v1/transfers", ({ request }) => {
      const nodeId = new URL(request.url).searchParams.get("nodeId");
      return HttpResponse.json({
        transfers: state.transfers.filter(
          ({ source, destination }) =>
            nodeId === null || source.nodeId === nodeId || destination.nodeId === nodeId,
        ),
      });
    }),
    http.get("*/api/v1/transfers/speeds", () => HttpResponse.json({ speeds: state.speeds })),
    http.get("*/api/v1/transfers/nodes/:nodeId/settings", () => HttpResponse.json(state.settings)),
    http.patch("*/api/v1/transfers/nodes/:nodeId/settings", async ({ request, params }) => {
      const body = SetTransferSettingsBody.parse(await request.json());
      writes.push({ nodeId: String(params["nodeId"]), body });
      state.settings = sendLimitOf(body.sendLimit ?? 0);
      return HttpResponse.json(state.settings);
    }),
  ];
  return { handlers, state, writes };
}
