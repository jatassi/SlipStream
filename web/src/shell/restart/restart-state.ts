import { useSyncExternalStore } from "react";

import { createStore } from "../store";

/** How often the page asks a restarting Node whether it is back, and when it gives up. */
export const pollInterval = 1000;
export const failAfter = 90_000;

/** `linkSince` is the Controller's `link.since` for the Node when the restart was asked for. */
export type RestartTarget = {
  readonly nodeId: string;
  readonly name: string;
  readonly self: boolean;
  readonly linkSince: string;
};

/** A restart that moves the port. `href` is set only when the browser reached the Node directly. */
export type Move = { readonly port: number; readonly href?: string };

type Waiting = {
  readonly before: string;
  readonly since: number;
  readonly logsDir: string;
  readonly move?: Move;
};

/**
 * The browser's own Node restarting. The page waits for a `startedAt` later
 * than `before`, fails after `failAfter`, and reloads once it is back.
 */
export type SelfRestart =
  | ({ readonly phase: "waiting" } & Waiting)
  | ({ readonly phase: "failed" } & Waiting)
  | { readonly phase: "back" };

/**
 * A Worker's restart. `at` is the browser's clock and `linkSince` the
 * Controller's, so the two are never compared.
 */
type WorkerRestart = { readonly at: number; readonly linkSince: string };

export type RestartState = {
  readonly asking: RestartTarget | null;
  readonly self: SelfRestart | null;
  readonly workers: ReadonlyMap<string, WorkerRestart>;
};

export type RestartEvent =
  | { readonly type: "ask"; readonly target: RestartTarget }
  | { readonly type: "dismiss" }
  | {
      readonly type: "accepted";
      readonly target: RestartTarget;
      readonly startedAt: string;
      readonly at: number;
      readonly logsDir: string;
      readonly move?: Move;
    }
  | { readonly type: "expire"; readonly nodeId: string; readonly at: number }
  | { readonly type: "back"; readonly nodeId: string }
  | { readonly type: "answered"; readonly startedAt: string }
  | { readonly type: "tick"; readonly at: number }
  | { readonly type: "retry"; readonly at: number };

export const idle: RestartState = { asking: null, self: null, workers: new Map() };

const later = (next: string, before: string) => Date.parse(next) > Date.parse(before);

function accept(state: RestartState, event: Extract<RestartEvent, { type: "accepted" }>) {
  const { target, startedAt, at, logsDir, move } = event;
  if (!target.self) {
    const workers = new Map(state.workers).set(target.nodeId, { at, linkSince: target.linkSince });
    return { ...state, asking: null, workers };
  }
  const waiting: SelfRestart = {
    phase: "waiting",
    before: startedAt,
    since: at,
    logsDir,
    ...(move && { move }),
  };
  return { ...state, asking: null, self: waiting };
}

type SelfEvent = Extract<RestartEvent, { type: "answered" | "tick" | "retry" }>;

function onSelf(self: SelfRestart | null, event: SelfEvent): SelfRestart | null {
  if (self === null || self.phase === "back") {
    return self;
  }
  if (event.type === "answered") {
    return later(event.startedAt, self.before) ? { phase: "back" } : self;
  }
  if (event.type === "tick") {
    return self.phase === "waiting" && event.at - self.since >= failAfter
      ? { ...self, phase: "failed" }
      : self;
  }
  return self.phase === "failed" ? { ...self, phase: "waiting", since: event.at } : self;
}

export function restartReducer(state: RestartState, event: RestartEvent): RestartState {
  if (event.type === "ask") {
    return state.self === null ? { ...state, asking: event.target } : state;
  }
  if (event.type === "dismiss") {
    return state.asking === null ? state : { ...state, asking: null };
  }
  if (event.type === "accepted") {
    return accept(state, event);
  }
  if (event.type === "expire" || event.type === "back") {
    const asked = state.workers.get(event.nodeId);
    if (asked === undefined || (event.type === "expire" && asked.at !== event.at)) {
      return state;
    }
    const workers = new Map(state.workers);
    workers.delete(event.nodeId);
    return { ...state, workers };
  }
  const self = onSelf(state.self, event);
  return self === state.self ? state : { ...state, self };
}

/**
 * A Worker restarts from the accepted request until its link is up with a
 * new since, which a `back` then clears, or until its `expire` after
 * `failAfter`.
 */
export function isWorkerRestarting(
  state: RestartState,
  node: { readonly id: string; readonly link: { readonly state: string; readonly since: string } },
): boolean {
  const asked = state.workers.get(node.id);
  return asked !== undefined && !(node.link.state === "up" && node.link.since !== asked.linkSince);
}

const defaultPorts: Readonly<Record<string, string>> = { "http:": "80", "https:": "443" };

/**
 * Where the page can follow a port move. A browser that reached the Node on
 * its bound port can follow it to the same host on the next port; behind a
 * proxy or a port mapping the browser's port isn't the Node's, so there is
 * no link. A next port the Node already holds is no move: a Node configured
 * for any port reports the one it bound as its next.
 */
export function moveFor(
  nextPort: number | undefined,
  boundPort: number,
  location: URL,
): Move | undefined {
  if (nextPort === undefined || nextPort === boundPort) {
    return undefined;
  }
  const port = location.port === "" ? defaultPorts[location.protocol] : location.port;
  if (port !== String(boundPort)) {
    return { port: nextPort };
  }
  const next = new URL(location);
  next.port = String(nextPort);
  return { port: nextPort, href: next.href };
}

export const restartStore = createStore(restartReducer, idle);

export function askRestart(node: {
  readonly id: string;
  readonly name: string;
  readonly self: boolean;
  readonly link: { readonly since: string };
}) {
  restartStore.dispatch({
    type: "ask",
    target: { nodeId: node.id, name: node.name, self: node.self, linkSince: node.link.since },
  });
}

export const useRestartState = (): RestartState =>
  useSyncExternalStore(restartStore.subscribe, restartStore.getSnapshot);
