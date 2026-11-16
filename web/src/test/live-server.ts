import { HttpResponse, http, sse } from "msw";
import { expect } from "vitest";

import type { IdentityAccount } from "../api/gen/model";
import { getGetSessionQueryKey } from "../api/gen/query/identity/identity";

import { type LiveMap, ada, problem } from "./handlers";
import { type renderApp, settled } from "./render-app";

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
  let account: IdentityAccount = ada;
  const handlers = [
    http.get("*/api/v1/session", () =>
      signedIn
        ? HttpResponse.json({ account, pinRequired: false, keyPrompt: "ask" })
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
    become(next: IdentityAccount) {
      account = next;
    },
  };
}

// The stream reports open before it reads the server's opening live.resync,
// which would invalidate anything seeded in between. The shell's guard loads
// the session before the stream connects, so the resync always marks it
// invalid, and only the shell mounting refetches it. A fetch still in flight
// when a test seeds its key would land after the seed and read as a refetch.
// A file's first page load compiles that page's modules, which on a loaded
// runner takes several seconds.
export async function synced(app: ReturnType<typeof renderApp>) {
  await expect
    .poll(() => {
      const session = app.queryClient.getQueryState(getGetSessionQueryKey());
      return session !== undefined && (session.isInvalidated || session.dataUpdateCount > 1);
    })
    .toBe(true);
  await expect.poll(() => settled(app), { timeout: 10_000 }).toBe(true);
}

// seeded caches an empty answer under each key and reads which were since
// invalidated, or refetched because something observes them.
export function seeded(
  { queryClient }: ReturnType<typeof renderApp>,
  keys: Record<string, readonly unknown[]>,
) {
  const updates = new Map<string, number | undefined>();
  for (const [name, key] of Object.entries(keys)) {
    queryClient.setQueryData(key, {});
    updates.set(name, queryClient.getQueryState(key)?.dataUpdateCount);
  }
  return () =>
    Object.fromEntries(
      Object.entries(keys).map(([name, key]) => {
        const state = queryClient.getQueryState(key);
        return [
          name,
          state !== undefined &&
            (state.isInvalidated || state.dataUpdateCount !== updates.get(name)),
        ];
      }),
    );
}
