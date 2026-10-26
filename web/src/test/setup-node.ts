import { HttpResponse, http } from "msw";
import { beforeEach } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { PairingWorkerClaim, ServingWorker, SystemStatus } from "../api/gen/model";
import { getGetServingMockHandler } from "../api/gen/query/serving/serving.msw";
import { writePref } from "../shell/prefs";
import { ada, noPasskeys, problem } from "./handlers";

export const rightCode = "7K2MQ9XD4HBT";

export const signedIn = {
  account: ada,
  pinRequired: false,
  keyPrompt: "ask",
  firstRun: "open",
  passkeyPrompt: false,
};

const seedbox: ServingWorker = {
  name: "seedbox",
  controller: "home",
  fingerprint: "7Q2M-0K4D-91XB-3FHA",
  holding: [],
  seeding: 0,
};

export type Node = {
  claim: SystemStatus["claim"];
  removed?: SystemStatus["removed"];
  join: PairingWorkerClaim | null;
  headers: { path: string; code: string | null }[];
};

export const wrongCode = () => problem(403, "setup_code_wrong", "The Setup code is wrong");

/** An Unclaimed Node with the Setup code rightCode, recording the code every Setup call sent. */
export function unclaimedNode(removed?: SystemStatus["removed"]) {
  const node: Node = { claim: "unclaimed", join: null, headers: [], ...(removed && { removed }) };
  const setupCall = (request: Request) => {
    const code = request.headers.get("X-Setup-Code");
    node.headers.push({ path: new URL(request.url).pathname, code });
    return code === rightCode;
  };
  const handlers = [
    http.get("*/api/v1/session", () =>
      node.claim === "controller"
        ? HttpResponse.json(signedIn)
        : problem(401, "signed_out", "Sign in to continue"),
    ),
    http.get("*/api/v1/status", () =>
      HttpResponse.json({
        claim: node.claim,
        passkeys: noPasskeys,
        ...(node.removed && { removed: node.removed }),
      }),
    ),
    http.post("*/api/v1/setup/code", ({ request }) =>
      setupCall(request) ? new HttpResponse(null, { status: 204 }) : wrongCode(),
    ),
    http.get("*/api/v1/setup/worker", ({ request }) => {
      if (!setupCall(request)) {
        return wrongCode();
      }
      return node.join === null
        ? problem(404, "no_join_pending", "No Join has awaited its Controller's dial")
        : HttpResponse.json(node.join);
    }),
    getGetServingMockHandler(seedbox),
  ];
  return { node, handlers };
}

export const setupCode = () => page.getByLabelText("Setup code");
export const startNew = () => page.getByRole("button", { name: "Start a new SlipStream" });
export const joinExisting = () => page.getByRole("button", { name: "Join an existing SlipStream" });

export async function typeCode(code: string) {
  await userEvent.type(setupCode(), code);
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
}

export async function fillStart(pinAgain: string) {
  await userEvent.fill(page.getByLabelText("Email"), "ada@example.test");
  await userEvent.fill(page.getByLabelText("Display name"), "Ada Okafor");
  await userEvent.click(page.getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard("1234");
  await userEvent.click(page.getByRole("textbox", { name: "PIN again", exact: true }));
  await userEvent.keyboard(pinAgain);
  await userEvent.click(page.getByRole("button", { name: "Create Admin" }));
}

export async function join(code = "SLP1-AAAA") {
  await userEvent.click(joinExisting());
  await userEvent.fill(page.getByLabelText("Join Code"), code);
  await userEvent.click(page.getByRole("button", { name: "Join", exact: true }));
}

export function claiming(answer: () => Response) {
  const server = unclaimedNode();
  const claims: unknown[] = [];
  const handlers = [
    http.post("*/api/v1/setup/controller", async ({ request }) => {
      claims.push({ code: request.headers.get("X-Setup-Code"), body: await request.json() });
      const response = answer();
      if (response.status === 201) {
        server.node.claim = "controller";
      }
      return response;
    }),
    ...server.handlers,
  ];
  return { claims, handlers };
}

export const awaiting: PairingWorkerClaim = {
  state: "awaiting-dial",
  nodeAddress: "SLN1-8F2KQ9XD4HBT",
  until: new Date(Date.now() + 10 * 60 * 1000).toISOString(),
};

export function joining(answer: (node: Node) => Response) {
  const server = unclaimedNode();
  const bodies: unknown[] = [];
  const handlers = [
    http.post("*/api/v1/setup/worker", async ({ request }) => {
      bodies.push(await request.json());
      return answer(server.node);
    }),
    ...server.handlers,
  ];
  return { node: server.node, bodies, handlers };
}

export function forgetKeptCode() {
  beforeEach(() => {
    writePref("setupCode", "");
  });
}
