import { HttpResponse, http, sse } from "msw";
import { afterEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";

import { getGetNodeQueryKey, getListNodesQueryKey } from "../api/gen/query/nodes/nodes";
import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import {
  getGetGrabQueryKey,
  getListGrabsQueryKey,
  getListSearchesQueryKey,
} from "../api/gen/query/grabs/grabs";
import { getGetSystemQueryKey } from "../api/gen/query/system/system";
import {
  getGetWorkQueryKey,
  getListWorksQueryKey,
  getSearchCatalogQueryKey,
} from "../api/gen/query/works/works";
import { type LiveMap, ada, problem } from "../test/handlers";
import { renderApp } from "../test/render-app";
import { worker } from "../test/worker";

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

function liveServer() {
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

function countRequests() {
  const counts = new Map<string, number>();
  const count = ({ request }: { request: Request }) => {
    const key = `${request.method} ${new URL(request.url).pathname}`;
    counts.set(key, (counts.get(key) ?? 0) + 1);
  };
  worker.events.on("request:start", count);
  return (key: string) => counts.get(key) ?? 0;
}

const liveState = () => page.getByText(/^Live updates: /u);

afterEach(() => {
  worker.events.removeAllListeners();
  vi.useRealTimers();
});

test("identity.sessions-changed refetches the session once", async () => {
  const server = liveServer();
  const requests = countRequests();
  renderApp({ path: "/", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  await expect.element(liveState()).toHaveTextContent("Live updates: on");
  await expect.poll(() => requests("GET /api/v1/session")).toBe(2);

  server.clients[0]?.send({ event: "identity.sessions-changed", data: { account: ada.id } });
  await expect.poll(() => requests("GET /api/v1/session")).toBe(3);
});

test("nodes.changed invalidates the Node list and that Node", async () => {
  const server = liveServer();
  const { queryClient } = renderApp({ path: "/", handlers: server.handlers });
  await expect.element(liveState()).toHaveTextContent("Live updates: on");
  const seedbox = "6a1f0c2e-8d4b-4f7a-9e3c-2b5d7f9a1c4e";
  const home = "0f9e8d7c-6b5a-4c3d-8e2f-1a0b9c8d7e6f";
  const keys = {
    list: getListNodesQueryKey(),
    seedbox: getGetNodeQueryKey(seedbox),
    home: getGetNodeQueryKey(home),
  };
  for (const key of Object.values(keys)) {
    queryClient.setQueryData(key, {});
  }
  const invalidated = () =>
    Object.fromEntries(
      Object.entries(keys).map(([name, key]) => [
        name,
        queryClient.getQueryState(key)?.isInvalidated,
      ]),
    );

  server.clients[0]?.send({ event: "nodes.changed", data: { nodeId: seedbox } });
  await expect.poll(invalidated).toEqual({ list: true, seedbox: true, home: false });
});

const orchard = "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03";
const harvest = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";
const grab = "7e9a1c3d-5f7b-4d9e-8a1c-5e7f9b1d3a4c";
const otherGrab = "9a1c3e5f-7b9d-4f1a-8c3e-7a9b1d3f5c6e";

function seeded(
  { queryClient }: ReturnType<typeof renderApp>,
  keys: Record<string, readonly unknown[]>,
) {
  for (const key of Object.values(keys)) {
    queryClient.setQueryData(key, {});
  }
  return () =>
    Object.fromEntries(
      Object.entries(keys).map(([name, key]) => [
        name,
        queryClient.getQueryState(key)?.isInvalidated,
      ]),
    );
}

test("works.changed invalidates the Works, that Work, its cells and every catalog search", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/system", handlers: server.handlers });
  await expect.element(liveState()).toHaveTextContent("Live updates: on");
  const invalidated = seeded(app, {
    list: getListWorksQueryKey(),
    work: getGetWorkQueryKey(orchard),
    cells: getGetWorkCellsQueryKey(orchard),
    otherCells: getGetWorkCellsQueryKey(harvest),
    search: getSearchCatalogQueryKey("cinema", { query: "orchard" }),
  });

  server.clients[0]?.send({ event: "works.changed", data: { workId: orchard } });
  await expect.poll(invalidated).toEqual({
    list: true,
    work: true,
    cells: true,
    otherCells: false,
    search: true,
  });
});

test("targets.changed invalidates every Work's cells and no Work", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/system", handlers: server.handlers });
  await expect.element(liveState()).toHaveTextContent("Live updates: on");
  const invalidated = seeded(app, {
    orchard: getGetWorkCellsQueryKey(orchard),
    harvest: getGetWorkCellsQueryKey(harvest),
    work: getGetWorkQueryKey(orchard),
  });

  server.clients[0]?.send({ event: "targets.changed", data: {} });
  await expect.poll(invalidated).toEqual({ orchard: true, harvest: true, work: false });
});

test("files.changed invalidates only that Work's cells", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/system", handlers: server.handlers });
  await expect.element(liveState()).toHaveTextContent("Live updates: on");
  const invalidated = seeded(app, {
    orchard: getGetWorkCellsQueryKey(orchard),
    harvest: getGetWorkCellsQueryKey(harvest),
  });

  server.clients[0]?.send({ event: "files.changed", data: { workId: orchard } });
  await expect.poll(invalidated).toEqual({ orchard: true, harvest: false });
});

test("grabs.changed invalidates the Grabs, that Grab and only that Work's cells", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/system", handlers: server.handlers });
  await expect.element(liveState()).toHaveTextContent("Live updates: on");
  const invalidated = seeded(app, {
    grabs: getListGrabsQueryKey(),
    grab: getGetGrabQueryKey(grab),
    other: getGetGrabQueryKey(otherGrab),
    orchard: getGetWorkCellsQueryKey(orchard),
    harvest: getGetWorkCellsQueryKey(harvest),
  });

  server.clients[0]?.send({ event: "grabs.changed", data: { grabId: grab, workId: orchard } });
  await expect
    .poll(invalidated)
    .toEqual({ grabs: true, grab: true, other: false, orchard: true, harvest: false });
});

test("grabs.search-changed invalidates only that Work's searches", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/system", handlers: server.handlers });
  await expect.element(liveState()).toHaveTextContent("Live updates: on");
  const invalidated = seeded(app, {
    orchard: getListSearchesQueryKey(orchard),
    harvest: getListSearchesQueryKey(harvest),
  });

  server.clients[0]?.send({
    event: "grabs.search-changed",
    data: { searchId: grab, workId: orchard },
  });
  await expect.poll(invalidated).toEqual({ orchard: true, harvest: false });
});

test("live.resync refetches every active query once", async () => {
  const server = liveServer();
  const requests = countRequests();
  renderApp({ path: "/system", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "This Node" })).toBeVisible();
  const reads = () => ({
    system: requests("GET /api/v1/system"),
    session: requests("GET /api/v1/session"),
  });
  await expect.poll(reads).toEqual({ system: 1, session: 2 });

  server.clients[0]?.send({ event: "live.resync", data: {} });
  await expect.poll(reads).toEqual({ system: 2, session: 3 });
});

test("live.session-ended clears this tab and every other tab", async () => {
  const server = liveServer();
  const first = renderApp({ path: "/system", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "This Node" })).toBeVisible();
  const second = renderApp({ path: "/" });
  await expect.poll(() => server.clients.length).toBe(2);

  server.signOut();
  server.clients[0]?.send({ event: "live.session-ended", data: { reason: "signed-out" } });
  await expect.poll(() => first.router.state.location.href).toBe("/sign-in?reason=signed-out");
  await expect.poll(() => second.router.state.location.href).toBe("/sign-in?reason=signed-out");
  expect(first.queryClient.getQueryData(getGetSystemQueryKey())).toBeUndefined();
  expect(second.queryClient.getQueryData(getGetSystemQueryKey())).toBeUndefined();
});

test("an event that fails to parse is reported once and resyncs", async () => {
  const server = liveServer();
  const requests = countRequests();
  const reported = vi.spyOn(globalThis, "reportError").mockImplementation(() => {});
  renderApp({ path: "/system", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "This Node" })).toBeVisible();

  server.clients[0]?.dispatchEvent(
    new MessageEvent("identity.sessions-changed", { data: JSON.stringify({ account: 42 }) }),
  );
  await expect.poll(() => requests("GET /api/v1/system")).toBe(2);
  expect(reported).toHaveBeenCalledOnce();
});

test("a refused stream with an expired session goes to sign-in", async () => {
  const server = liveServer();
  const { router } = renderApp({
    path: "/",
    handlers: [
      http.get("*/api/v1/events", () => {
        server.signOut();
        return problem(401, "signed_out", "Sign in to continue");
      }),
      ...server.handlers,
    ],
  });
  await expect.poll(() => router.state.location.href).toBe("/sign-in?reason=expired&redirect=%2F");
  await expect.element(page.getByText("Your session ended. Sign in again.")).toBeVisible();
});

test("a refused stream with a live session reopens after a second", async () => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"], shouldAdvanceTime: true });
  const server = liveServer();
  renderApp({
    path: "/",
    handlers: [
      http.get("*/api/v1/events", () => problem(500, "internal", "Something failed"), {
        once: true,
      }),
      ...server.handlers,
    ],
  });
  await expect.element(liveState()).toHaveTextContent("Live updates: reconnecting");
  expect(server.clients).toHaveLength(0);

  await vi.advanceTimersByTimeAsync(1000);
  await expect.element(liveState()).toHaveTextContent("Live updates: on");
  expect(server.clients).toHaveLength(1);
});

test("a hidden tab releases its stream after 10 seconds and reopens when shown", async () => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"], shouldAdvanceTime: true });
  let visibility: DocumentVisibilityState = "visible";
  vi.spyOn(document, "visibilityState", "get").mockImplementation(() => visibility);
  const show = (next: DocumentVisibilityState) => {
    visibility = next;
    document.dispatchEvent(new Event("visibilitychange"));
  };
  const server = liveServer();
  renderApp({ path: "/", handlers: server.handlers });
  await expect.element(liveState()).toHaveTextContent("Live updates: on");

  show("hidden");
  await vi.advanceTimersByTimeAsync(10_000);
  await expect.element(liveState()).toHaveTextContent("Live updates: paused");
  await expect.poll(() => server.closed).toEqual([0]);

  show("visible");
  await expect.element(liveState()).toHaveTextContent("Live updates: on");
  expect(server.clients).toHaveLength(2);
});
