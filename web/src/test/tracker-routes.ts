import { HttpResponse, http } from "msw";

import type {
  DownloadclientsClientCard,
  DownloadclientsExternalClient,
  TrackersChoice,
  TrackersRoute,
  TrackersTarget,
} from "../api/gen/model";
import { SetTrackerRouteBody } from "../api/gen/zod/trackers/trackers.zod";
import { downloadClients, homeCard, seedboxCard } from "./download-clients";
import { attic, withExternals } from "./external-clients";
import { homeId } from "./nodes";
import { trackersServer } from "./trackers";

/** seedpool's one home, home, whose Grabs go to its built-in client by default. */
export const homeRoute: TrackersRoute = {
  homeNodeId: homeId,
  to: { nodeId: homeId },
  default: { nodeId: homeId },
  stored: null,
  skipped: false,
};

/** home with Transmission, and seedbox: the clients a route can pick. */
export const routingCards: DownloadclientsClientCard[] = [
  withExternals(homeCard, attic),
  seedboxCard,
];

export type RouteWrite =
  | { method: "PUT"; nodeId: string; body: TrackersChoice }
  | { method: "DELETE"; nodeId: string };

export type RouteRefusal = { status: number; code: string; title: string; detail?: string };

const refused = (refusal: RouteRefusal) =>
  HttpResponse.json(refusal, {
    status: refusal.status,
    headers: { "Content-Type": "application/problem+json" },
  });

function targetOf(
  externals: readonly DownloadclientsExternalClient[],
  { clientId, builtInOn }: TrackersChoice,
): TrackersTarget {
  const client = externals.find(({ id }) => id === clientId);
  return client === undefined
    ? { nodeId: builtInOn ?? "" }
    : { nodeId: client.nodeId, clientId: client.id };
}

/**
 * The Trackers server with seedpool's routes and the Download Clients
 * list. A PUT stores the pick and a DELETE drops it, each recorded in
 * writes, unless refuse gives the problem to answer.
 */
export function routingServer({
  routes = [homeRoute],
  cards = routingCards,
  refuse,
}: { routes?: TrackersRoute[]; cards?: DownloadclientsClientCard[]; refuse?: RouteRefusal } = {}) {
  const state = { routes: [...routes] };
  const writes: RouteWrite[] = [];
  const externals = cards.flatMap((card) => card.externals);
  const base = trackersServer();
  const change = (nodeId: string, edit: (route: TrackersRoute) => TrackersRoute) => {
    state.routes = state.routes.map((route) => (route.homeNodeId === nodeId ? edit(route) : route));
    return HttpResponse.json({ routes: state.routes });
  };
  const handlers = [
    http.get("*/api/v1/trackers/:id/routes", () => HttpResponse.json({ routes: state.routes })),
    http.put("*/api/v1/trackers/:id/routes/:nodeId", async ({ params, request }) => {
      const nodeId = String(params["nodeId"]);
      const body = SetTrackerRouteBody.parse(await request.json());
      writes.push({ method: "PUT", nodeId, body });
      if (refuse !== undefined) {
        return refused(refuse);
      }
      return change(nodeId, (route) => ({
        ...route,
        stored: body,
        to: targetOf(externals, body),
        skipped: false,
      }));
    }),
    http.delete("*/api/v1/trackers/:id/routes/:nodeId", ({ params }) => {
      const nodeId = String(params["nodeId"]);
      writes.push({ method: "DELETE", nodeId });
      return change(nodeId, (route) => ({
        ...route,
        stored: null,
        to: route.default,
        skipped: false,
      }));
    }),
    ...downloadClients({ cards }).handlers,
    ...base.handlers,
  ];
  return { handlers, state, writes, trackers: base.state };
}
