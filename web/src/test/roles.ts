import type { QueryClient } from "@tanstack/react-query";
import { HttpResponse, http } from "msw";

import type {
  NodesBoardNode,
  NodesOutcome,
  NodesRoleBoard,
  NodesRoleGroup,
} from "../api/gen/model";
import { getGetRoleBoardQueryKey } from "../api/gen/query/nodes/nodes";
import { getGetRoleBoardMockHandler } from "../api/gen/query/nodes/nodes.msw";
import { NodesPlanChange, NodesWriteOffBody } from "../api/gen/zod/slipStream.schemas.zod";
import { refresh } from "../live/refresh";
import { homeId, seedboxId } from "./nodes";
import { worker } from "./worker";

export const groups: NodesRoleGroup[] = [
  {
    group: "acquire",
    label: "Acquire",
    roles: [
      { role: "searching", label: "Searching", pinned: false },
      { role: "downloading", label: "Downloading", probe: "engine-starts", pinned: false },
    ],
  },
  {
    group: "store",
    label: "Store",
    roles: [{ role: "storage", label: "Storage", probe: "data-writable", pinned: false }],
  },
  {
    group: "process",
    label: "Process",
    roles: [
      { role: "converting", label: "Converting", pinned: false },
      { role: "subtitles", label: "Subtitles", pinned: false },
    ],
  },
  {
    group: "connect",
    label: "Connect",
    roles: [
      { role: "metadata", label: "Metadata", pinned: false },
      { role: "media-server-link", label: "Media Server link", pinned: false },
      { role: "notifying", label: "Notifying", pinned: false },
    ],
  },
  {
    group: "reach",
    label: "Reach",
    roles: [
      { role: "relaying", label: "Relaying", pinned: false },
      { role: "serving", label: "Serving", probe: "listener-up", pinned: true },
    ],
  },
];

export const homeSeat: NodesBoardNode = {
  id: homeId,
  name: "home",
  controller: true,
  away: false,
  assigned: ["searching", "storage", "metadata", "notifying", "serving"],
  roles: [
    { role: "searching", state: "holding" },
    { role: "storage", state: "holding" },
    { role: "metadata", state: "holding" },
    { role: "notifying", state: "holding" },
    { role: "serving", state: "holding" },
  ],
  probes: [],
  suggested: ["searching", "downloading"],
};

export const seedboxSeat: NodesBoardNode = {
  id: seedboxId,
  name: "seedbox",
  controller: false,
  away: false,
  assigned: ["downloading"],
  roles: [
    {
      role: "downloading",
      state: "holding",
      held: { units: 3, noun: "torrents", bytes: 41 * 1024 ** 3, endless: true },
    },
  ],
  probes: [{ probe: "engine-starts", fit: "fits", at: "2026-10-12T10:00:00Z" }],
  suggested: ["searching", "downloading"],
};

export const board = (...nodes: NodesBoardNode[]): NodesRoleBoard => ({
  basis: "basis-1",
  groups,
  nodes,
  gestures: [],
});

export const boardIs = (shown: NodesRoleBoard) => getGetRoleBoardMockHandler(shown);

/** Serves a new Role board from here on and refreshes it, as a live event would. */
export async function boardBecomes(queryClient: QueryClient, shown: NodesRoleBoard) {
  worker.use(boardIs(shown));
  await refresh(queryClient, { queryKey: getGetRoleBoardQueryKey() });
}

export const outcome = (shown: Partial<NodesOutcome>): NodesOutcome => ({
  basis: "basis-1",
  checks: [],
  effects: [],
  forfeit: [],
  next: [],
  ...shown,
});

export function refusal(code: string, title: string, carried: NodesOutcome) {
  return HttpResponse.json(
    { status: 409, code, title, outcome: carried },
    { status: 409, headers: { "Content-Type": "application/problem+json" } },
  );
}

type Answer = NodesOutcome | (() => Response);

const answered = (answer: Answer) =>
  typeof answer === "function" ? answer() : HttpResponse.json(answer);

/** planRoles and setRoles answering in turn, with every change they were sent. */
export function rolesServer({ plans, sets }: { plans: Answer[]; sets: Answer[] }) {
  const planned: NodesPlanChange[] = [];
  const applied: NodesPlanChange[] = [];
  const handlers = [
    http.post("*/api/v1/nodes/roles/plan", async ({ request }) => {
      planned.push(NodesPlanChange.parse(await request.json()));
      return answered(plans[planned.length - 1] ?? outcome({}));
    }),
    http.put("*/api/v1/nodes/roles", async ({ request }) => {
      applied.push(NodesPlanChange.parse(await request.json()));
      return answered(sets[applied.length - 1] ?? outcome({}));
    }),
  ];
  return { planned, applied, handlers };
}

/** writeOff answering answer, with every body it was sent by Node. */
export function writeOffServer(answer: Answer) {
  const sent: { nodeId: string; body: NodesWriteOffBody }[] = [];
  const handler = http.post("*/api/v1/nodes/:id/write-offs", async ({ request, params }) => {
    sent.push({
      nodeId: String(params["id"]),
      body: NodesWriteOffBody.parse(await request.json()),
    });
    return answered(answer);
  });
  return { sent, handler };
}
