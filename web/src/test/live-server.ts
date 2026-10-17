import { HttpResponse, http, sse } from "msw";

import { type LiveMap, ada, problem } from "./handlers";
import type { renderApp } from "./render-app";

type Client = Parameters<Parameters<typeof sse<LiveMap>>[1]>[0]["client"];

// MSW's sse has no public close signal and the request's signal never aborts
// through the service worker; the client's emitter is what MSW's own logger uses.
function onClose(client: Client, listener: () => void) {
  // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- reaching MSW's internal emitter
  const emitters = client as unknown as Record<
    symbol,
    { on: (type: "close", listener: () => void) => void }
  >;
  emitters[Symbol.for("kClientEmitter")]?.on("close", listener);
}

export function liveServer() {
  const clients: Client[] = [];
  const closed: number[] = [];
  let signedIn = true;
  const handlers = [
    http.get("*/api/v1/session", () =>
      signedIn
        ? HttpResponse.json({ account: ada })
        : problem(401, "signed_out", "Sign in to continue"),
    ),
    sse<LiveMap>("*/api/v1/events", ({ client }) => {
      const index = clients.push(client) - 1;
      onClose(client, () => {
        closed.push(index);
      });
      client.send({ event: "live.resync", data: {} });
    }),
  ];
  return {
    handlers,
    clients,
    closed,
    signOut() {
      signedIn = false;
    },
  };
}

// seeded caches an empty answer under each key and reads which were since
// invalidated.
export function seeded(
  { queryClient }: ReturnType<typeof renderApp>,
  keys: Record<string, readonly unknown[]>,
) {
  for (const key of Object.values(keys)) {
    queryClient.setQueryData(key, {});
  }
  const invalidated = new Set<string>();
  queryClient.getQueryCache().subscribe((event) => {
    if (event.type === "updated" && event.action.type === "invalidate") {
      invalidated.add(event.query.queryHash);
    }
  });
  return () =>
    Object.fromEntries(
      Object.entries(keys).map(([name, key]) => [
        name,
        invalidated.has(
          queryClient.getQueryCache().find({ queryKey: key, exact: true })?.queryHash ?? "",
        ),
      ]),
    );
}
