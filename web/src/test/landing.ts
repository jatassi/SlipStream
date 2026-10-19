import { HttpResponse, http } from "msw";

import type { NodesBoardNode, NodesNode } from "../api/gen/model";
import { home, homeId, nodesAre, seedbox, seedboxId, topologyIs, topologyOf } from "./nodes";
import { board, boardIs, homeSeat, seedboxSeat } from "./roles";

export const vpsId = "7d9f1b3c-5e7a-4c9e-8b1d-3f5a7c9e1b2d";

/** seedbox offline, draining the 3 torrents it still seeds, on a network share. */
export const drainingSeedbox: NodesNode = {
  ...seedbox,
  roles: [
    {
      role: "downloading",
      state: "draining",
      held: { units: 3, noun: "torrents", bytes: 41 * 1024 ** 3, endless: true },
    },
  ],
  probes: [
    {
      probe: "engine-starts",
      fit: "weak",
      at: "2026-10-12T10:00:00Z",
      reason: "downloads_on_network_share",
      words: "The downloads folder is on a network share.",
    },
  ],
};

/** vps online, dialed by home. */
export const vps: NodesNode = {
  ...seedbox,
  id: vpsId,
  name: "vps",
  key: "SHA256:vps-key",
  link: { state: "up", since: "2026-10-12T11:00:00Z" },
  standing: "in-step",
  dial: { by: "controller", controllerDials: ["203.0.113.9:8485"] },
  roles: [{ role: "searching", state: "holding" }],
  probes: [],
};

const vpsSeat: NodesBoardNode = {
  id: vpsId,
  name: "vps",
  controller: false,
  away: false,
  assigned: ["searching"],
  roles: [{ role: "searching", state: "holding" }],
  probes: [],
  suggested: [],
};

/** listNodes, Topology and the role board for home, seedbox and vps. */
export const threeNodes = [
  nodesAre(home, drainingSeedbox, vps),
  topologyIs(
    topologyOf([home, drainingSeedbox, vps], {
      sentence: "seedbox is offline.",
      rtt: 24,
    }),
  ),
  boardIs(board(homeSeat, seedboxSeat, vpsSeat)),
];

export const ids = { home: homeId, seedbox: seedboxId, vps: vpsId };

/** renameNode and removeNode answering in turn, with what each was sent. */
export function manageServer({
  renames = [],
  removes = [],
}: {
  renames?: (() => Response)[];
  removes?: (() => Response)[];
}) {
  const renamed: { nodeId: string; body: unknown }[] = [];
  const removed: { nodeId: string; writeOff: string | null }[] = [];
  const handlers = [
    http.patch("*/api/v1/nodes/:id", async ({ params, request }) => {
      renamed.push({ nodeId: String(params["id"]), body: await request.json() });
      return renames[renamed.length - 1]?.() ?? HttpResponse.json(home);
    }),
    http.delete("*/api/v1/nodes/:id", ({ params, request }) => {
      removed.push({
        nodeId: String(params["id"]),
        writeOff: new URL(request.url).searchParams.get("writeOff"),
      });
      return removes[removed.length - 1]?.() ?? new HttpResponse(null, { status: 204 });
    }),
  ];
  return { renamed, removed, handlers };
}
