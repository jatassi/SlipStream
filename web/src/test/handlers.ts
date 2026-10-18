import { HttpResponse, type RequestHandler, http, sse } from "msw";
import { onTestFinished } from "vitest";

import type { IdentityAccount, SystemPasskeys } from "../api/gen/model";
import { getGetSessionMockHandler } from "../api/gen/query/identity/identity.msw";
import { getGetCountsMockHandler } from "../api/gen/query/shell/shell.msw";
import { getGetStatusMockHandler } from "../api/gen/query/system/system.msw";
import { getListOpenTaskRunsMockHandler } from "../api/gen/query/tasks/tasks.msw";
import type { LiveEventName, PayloadOf } from "../live/types";
import { home, homeId, nodesAre, topologyIs, topologyOf } from "./nodes";
import { board, boardIs, homeSeat } from "./roles";

const generated = import.meta.glob<Record<string, () => RequestHandler[]>>(
  "../api/gen/query/**/*.msw.ts",
  { eager: true },
);

const tagMocks = Object.entries(generated).flatMap(([path, module]) => {
  const tag = path
    .replaceAll(/^.*\/|\.msw\.ts$/gu, "")
    .split(/[^a-z0-9]+/iu)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join("");
  const handlers = module[`get${tag}Mock`];
  if (!handlers) {
    throw new Error(`${path} exports no get${tag}Mock`);
  }
  return handlers();
});

export const ada: IdentityAccount = {
  id: "0b6f3a8e-2c4d-4e1f-9a7b-5d8c6e2f1a3b",
  email: "ada@example.test",
  displayName: "Ada Okafor",
  admin: true,
};

export const grace: IdentityAccount = {
  id: "5c1d7e9a-3b2f-4a6c-8d0e-1f4a7b9c2e5d",
  email: "grace@example.test",
  displayName: "Grace Lindqvist",
  admin: false,
};

export const signedInAsGrace = getGetSessionMockHandler({
  account: grace,
  pinRequired: false,
  keyPrompt: "ask",
});

export function problem(status: number, code: string, title: string) {
  return HttpResponse.json(
    { status, code, title },
    { status, headers: { "Content-Type": "application/problem+json" } },
  );
}

/** The Overview of the Node every test talks to. */
export const homePage = `/nodes/${homeId}`;

export type LiveMap = { [N in LiveEventName]: PayloadOf<N> };

export const noPasskeys: SystemPasskeys = { available: false, reason: "plain_http" };

const controller = getGetStatusMockHandler({ claim: "controller", passkeys: noPasskeys });
const nothingMoving = getGetCountsMockHandler({ activity: 0, alerts: { nodes: null } });
const signedInAsAda = getGetSessionMockHandler({
  account: ada,
  pinRequired: false,
  keyPrompt: "ask",
});
const noTaskRunning = getListOpenTaskRunsMockHandler({ headerDelaySeconds: 5, runs: [] });
const resyncStream = sse<LiveMap>("*/api/v1/events", ({ client }) => {
  client.send({ event: "live.resync", data: {} });
});

export const baseHandlers: RequestHandler[] = [
  controller,
  signedInAsAda,
  resyncStream,
  nothingMoving,
  nodesAre(home),
  topologyIs(topologyOf([home], { sentence: "home is online." })),
  boardIs(board(homeSeat)),
  noTaskRunning,
  ...tagMocks,
];

export function pendingForever(): RequestHandler[] {
  let release!: () => void;
  const released = new Promise<void>((resolve) => {
    release = resolve;
  });
  onTestFinished(release);
  return [
    controller,
    signedInAsAda,
    resyncStream,
    http.all("*/api/v1/*", async () => {
      await released;
      return problem(503, "unavailable", "Released at the end of the test");
    }),
  ];
}

export const signedOut = http.get("*/api/v1/session", () =>
  problem(401, "signed_out", "Sign in to continue"),
);
