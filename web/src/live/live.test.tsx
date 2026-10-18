import type { QueryClient } from "@tanstack/react-query";
import { http } from "msw";
import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import {
  getGetWorkQueryKey,
  getListWorksQueryKey,
  getSearchCatalogQueryKey,
} from "../api/gen/query/works/works";
import { ada, grace, problem } from "../test/handlers";
import { liveServer, seeded, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { worker } from "../test/worker";

function countRequests() {
  const counts = new Map<string, number>();
  const count = ({ request }: { request: Request }) => {
    const path = new URL(request.url).pathname.replace(
      /^\/api\/v1\/nodes\/[^/]+\//u,
      "/api/v1/nodes/:id/",
    );
    const key = `${request.method} ${path}`;
    counts.set(key, (counts.get(key) ?? 0) + 1);
  };
  worker.events.on("request:start", count);
  return (key: string) => counts.get(key) ?? 0;
}

afterEach(() => {
  worker.events.removeAllListeners();
  vi.useRealTimers();
});

const nodeReads = (queryClient: QueryClient) =>
  queryClient
    .getQueryCache()
    .findAll({ predicate: ({ queryKey }) => String(queryKey[0]).startsWith("/api/v1/nodes") });

const overview = "GET /api/v1/nodes/:id/system";

test("identity.sessions-changed refetches the session once", async () => {
  const server = liveServer();
  const requests = countRequests();
  const app = renderApp({ path: "/", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  await expect.poll(() => app.live.getState()).toBe("open");
  await expect.poll(() => requests("GET /api/v1/session")).toBe(2);

  server.clients[0]?.send({ event: "identity.sessions-changed", data: { account: ada.id } });
  await expect.poll(() => requests("GET /api/v1/session")).toBe(3);
});

const orchard = "3c1e9a52-7b4d-4f0e-8a6c-2d5b9e1f7a03";
const harvest = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";

test("works.changed invalidates the Works, that Work, its cells and every catalog search", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
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
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
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
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await synced(app);
  const invalidated = seeded(app, {
    orchard: getGetWorkCellsQueryKey(orchard),
    harvest: getGetWorkCellsQueryKey(harvest),
  });

  server.clients[0]?.send({ event: "files.changed", data: { workId: orchard } });
  await expect.poll(invalidated).toEqual({ orchard: true, harvest: false });
});

test("live.resync refetches every active query once", async () => {
  const server = liveServer();
  const requests = countRequests();
  renderApp({ path: "/nodes", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "home" })).toBeVisible();
  const reads = () => ({
    system: requests(overview),
    session: requests("GET /api/v1/session"),
  });
  await expect.poll(reads).toEqual({ system: 1, session: 2 });

  server.clients[0]?.send({ event: "live.resync", data: {} });
  await expect.poll(reads).toEqual({ system: 2, session: 3 });
});

test("live.session-ended clears this tab and every other tab", async () => {
  const server = liveServer();
  const first = renderApp({ path: "/nodes", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "home" })).toBeVisible();
  const second = renderApp({ path: "/" });
  await expect.poll(() => server.clients.length).toBe(2);
  await expect.poll(() => second.queryClient.isFetching()).toBe(0);

  server.signOut();
  await second.queryClient.refetchQueries();
  server.clients[0]?.send({ event: "live.session-ended", data: { reason: "signed-out" } });
  await expect.poll(() => first.router.state.location.href).toBe("/sign-in?reason=signed-out");
  await expect.poll(() => second.router.state.location.href).toBe("/sign-in?reason=signed-out");
  expect(nodeReads(first.queryClient)).toEqual([]);
  expect(nodeReads(second.queryClient)).toEqual([]);
});

test("live.access-changed re-gates the tab, so a demoted Admin lands on the portal", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/nodes", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "home" })).toBeVisible();
  await expect.poll(() => app.live.getState()).toBe("open");

  server.become({ ...ada, admin: false });
  server.clients[0]?.dispatchEvent(new MessageEvent("live.access-changed", { data: "{}" }));
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  expect(app.router.state.location.href).toBe("/portal");
  await expect.element(page.getByRole("navigation", { name: "Main" })).not.toBeInTheDocument();
  expect(nodeReads(app.queryClient).map((query) => query.state.data)).toEqual([
    undefined,
    undefined,
  ]);
  await expect.poll(() => server.closed).toEqual([0]);
  await expect.poll(() => server.clients).toHaveLength(2);
  await expect.poll(() => app.live.getState()).toBe("open");
});

test("live.access-changed keeps a promoted Member in the portal they were in, with Admin links", async () => {
  const server = liveServer();
  server.become(grace);
  const app = renderApp({ path: "/portal", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  await expect.poll(() => app.live.getState()).toBe("open");

  server.become({ ...grace, admin: true });
  server.clients[0]?.dispatchEvent(new MessageEvent("live.access-changed", { data: "{}" }));
  await expect.poll(() => server.clients).toHaveLength(2);
  await userEvent.click(page.getByRole("button", { name: "Account" }));
  await expect.element(page.getByRole("menuitem", { name: "Back to Library" })).toBeVisible();
  expect(app.router.state.location.href).toBe("/portal");
});

test("an event that fails to parse is reported once and resyncs", async () => {
  const server = liveServer();
  const requests = countRequests();
  const reported = vi.spyOn(globalThis, "reportError").mockImplementation(() => {});
  renderApp({ path: "/nodes", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "home" })).toBeVisible();

  server.clients[0]?.dispatchEvent(
    new MessageEvent("identity.sessions-changed", { data: JSON.stringify({ account: 42 }) }),
  );
  await expect.poll(() => requests(overview)).toBe(2);
  expect(reported).toHaveBeenCalledOnce();
});

test("a nodes.traffic sample that fails to parse is reported and dropped, never resynced", async () => {
  const server = liveServer();
  const requests = countRequests();
  const reported = vi.spyOn(globalThis, "reportError").mockImplementation(() => {});
  renderApp({ path: "/nodes", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "home" })).toBeVisible();
  await expect.poll(() => requests(overview)).toBe(1);

  server.clients[0]?.dispatchEvent(
    new MessageEvent("nodes.traffic", { data: JSON.stringify({ at: "soon", links: 3 }) }),
  );
  server.clients[0]?.send({ event: "live.resync", data: {} });
  await expect.poll(() => requests(overview)).toBe(2);
  expect(reported).toHaveBeenCalledOnce();
});

test("a refused stream with an expired session goes to sign-in", async () => {
  const server = liveServer();
  const { router } = renderApp({
    path: "/library",
    handlers: [
      http.get("*/api/v1/events", () => {
        server.signOut();
        return problem(401, "signed_out", "Sign in to continue");
      }),
      ...server.handlers,
    ],
  });
  await expect
    .poll(() => router.state.location.href)
    .toBe("/sign-in?reason=expired&redirect=%2Flibrary");
  await expect.element(page.getByText("Your session ended. Sign in again.")).toBeVisible();
});

test("a refused stream with a live session reopens after a second", async () => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"], shouldAdvanceTime: true });
  const server = liveServer();
  const app = renderApp({
    path: "/",
    handlers: [
      http.get("*/api/v1/events", () => problem(500, "internal", "Something failed"), {
        once: true,
      }),
      ...server.handlers,
    ],
  });
  await expect.element(page.getByText("Reconnecting to SlipStream…")).toBeVisible();
  expect(server.clients).toHaveLength(0);

  await vi.advanceTimersByTimeAsync(1000);
  await expect.poll(() => app.live.getState()).toBe("open");
  await expect.element(page.getByText("Reconnecting to SlipStream…")).not.toBeInTheDocument();
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
  const app = renderApp({ path: "/", handlers: server.handlers });
  await expect.poll(() => app.live.getState()).toBe("open");

  show("hidden");
  await vi.advanceTimersByTimeAsync(10_000);
  await expect.poll(() => app.live.getState()).toBe("paused");
  await expect.poll(() => server.closed).toEqual([0]);

  show("visible");
  await expect.poll(() => app.live.getState()).toBe("open");
  expect(server.clients).toHaveLength(2);
});
