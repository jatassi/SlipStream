import { expect, test } from "vitest";

import { createStore } from "../store";
import {
  type RestartEvent,
  type RestartState,
  type RestartTarget,
  idle,
  isWorkerRestarting,
  moveFor,
  restartReducer,
} from "./restart-state";

const home: RestartTarget = { nodeId: "home-id", name: "home", self: true };
const seedbox: RestartTarget = { nodeId: "seedbox-id", name: "seedbox", self: false };
const before = "2026-10-16T09:00:00.123456789Z";
const after = "2026-10-16T09:00:07.5Z";

const run = (...events: RestartEvent[]): RestartState =>
  events.reduce((state, event) => restartReducer(state, event), idle);

const acceptedHome: RestartEvent = {
  type: "accepted",
  target: home,
  startedAt: before,
  at: 1_000,
  logsDir: "/data/home/logs",
};

test("asking opens the confirm for a Node, and dismissing closes it", () => {
  expect(run({ type: "ask", target: seedbox }).asking).toEqual(seedbox);
  expect(run({ type: "ask", target: seedbox }, { type: "dismiss" }).asking).toBeNull();
});

test("an accepted restart of the browser's Node waits until the Node answers with a later start", () => {
  const waiting = run({ type: "ask", target: home }, acceptedHome);
  expect(waiting).toEqual({
    asking: null,
    self: { phase: "waiting", before, since: 1_000, logsDir: "/data/home/logs" },
    workers: new Map(),
  });
  expect(restartReducer(waiting, { type: "answered", startedAt: before }).self?.phase).toBe(
    "waiting",
  );
  expect(restartReducer(waiting, { type: "answered", startedAt: after }).self).toEqual({
    phase: "back",
  });
});

test("after 90 seconds the wait fails with the log folder, and Try again waits again", () => {
  const waiting = run(acceptedHome, { type: "tick", at: 90_999 });
  expect(waiting.self?.phase).toBe("waiting");
  const failed = restartReducer(waiting, { type: "tick", at: 91_000 });
  expect(failed.self).toEqual({
    phase: "failed",
    before,
    since: 1_000,
    logsDir: "/data/home/logs",
  });
  const retried = restartReducer(failed, { type: "retry", at: 200_000 });
  expect(retried.self).toEqual({
    phase: "waiting",
    before,
    since: 200_000,
    logsDir: "/data/home/logs",
  });
  expect(restartReducer(retried, { type: "answered", startedAt: after }).self).toEqual({
    phase: "back",
  });
});

test("a second restart can't be asked for while the browser's Node restarts", () => {
  expect(run(acceptedHome, { type: "ask", target: seedbox }).asking).toBeNull();
});

test("a restart that moves the port keeps the move", () => {
  const moved = run({ ...acceptedHome, move: { port: 9000, href: "http://media.local:9000/" } });
  expect(moved.self).toEqual({
    phase: "waiting",
    before,
    since: 1_000,
    logsDir: "/data/home/logs",
    move: { port: 9000, href: "http://media.local:9000/" },
  });
});

test("a Worker restarts until its link is up again after the request", () => {
  const asked = Date.parse("2026-10-16T09:00:00Z");
  const state = run(
    { type: "ask", target: seedbox },
    { type: "accepted", target: seedbox, startedAt: before, at: asked, logsDir: "/srv/logs" },
  );
  expect(state.self).toBeNull();
  expect(state.asking).toBeNull();
  const seedboxAt = (linkState: string, since: string) =>
    isWorkerRestarting(state, { id: seedbox.nodeId, link: { state: linkState, since } });
  expect(seedboxAt("up", "2026-10-16T08:00:00Z")).toBe(true);
  expect(seedboxAt("down", "2026-10-16T09:00:02Z")).toBe(true);
  expect(seedboxAt("up", "2026-10-16T09:00:05Z")).toBe(false);
  expect(
    isWorkerRestarting(state, {
      id: "other",
      link: { state: "down", since: "2026-10-16T09:00:02Z" },
    }),
  ).toBe(false);
});

test("a port move links to the new port only when the browser reached the Node on its bound port", () => {
  expect(moveFor(9000, 8484, new URL("http://media.local:8484/nodes/abc/web"))).toEqual({
    port: 9000,
    href: "http://media.local:9000/nodes/abc/web",
  });
  expect(moveFor(9000, 8484, new URL("https://slipstream.example.com/nodes"))).toEqual({
    port: 9000,
  });
  expect(moveFor(9000, 80, new URL("http://[::1]/"))).toEqual({
    port: 9000,
    href: "http://[::1]:9000/",
  });
  expect(moveFor(undefined, 8484, new URL("http://media.local:8484/"))).toBeUndefined();
  expect(moveFor(8484, 8484, new URL("http://media.local:8484/"))).toBeUndefined();
});

test("the store notifies on a change and stays quiet on an event that changes nothing", () => {
  const store = createStore(restartReducer, idle);
  let heard = 0;
  store.subscribe(() => {
    heard += 1;
  });
  store.dispatch({ type: "dismiss" });
  store.dispatch({ type: "ask", target: home });
  store.dispatch({ type: "tick", at: 5 });
  expect(heard).toBe(1);
  expect(store.getSnapshot().asking).toEqual(home);
});
