import { type RequestHandler, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { NodesBoardNode, NodesNode } from "../../../api/gen/model";
import { problem } from "../../../test/handlers";
import { codeText, joinCodeServer, nodesFrom } from "../../../test/join-codes";
import { liveServer, synced } from "../../../test/live-server";
import { home } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { board, boardBecomes, boardIs, homeSeat, outcome, rolesServer } from "../../../test/roles";

const vpsId = "7d9f1b3c-5e7a-4c9e-8b1d-3f5a7c9e1b2d";

const vps: NodesNode = {
  id: vpsId,
  name: "vps",
  self: false,
  addresses: ["203.0.113.9:8485"],
  intentVersion: "00112233445566778899aabbccddeeff",
  key: "SHA256:vps-key",
  link: { state: "up", since: "2026-10-18T21:00:00Z" },
  standing: "in-step",
  dial: { by: "controller", controllerDials: ["203.0.113.9:8485"] },
  pairedAt: "2026-10-18T21:00:00Z",
  roles: [],
  probes: [{ probe: "engine-starts", fit: "fits", at: "2026-10-18T21:00:00Z", words: "" }],
};

const vpsSeat: NodesBoardNode = {
  id: vpsId,
  name: "vps",
  controller: false,
  away: false,
  assigned: [],
  roles: [],
  probes: [],
  suggested: ["searching", "downloading"],
};

const dialog = () => page.getByRole("dialog", { name: "Add a Node" });
const button = (name: string) => dialog().getByRole("button", { name, exact: true });

function addNode({
  path = "/nodes",
  codes = joinCodeServer(),
  handlers = [],
}: { path?: string; codes?: ReturnType<typeof joinCodeServer>; handlers?: RequestHandler[] } = {}) {
  const live = liveServer();
  let nodes: NodesNode[] = [home];
  const app = renderApp({
    path,
    handlers: [...handlers, ...live.handlers, ...codes.handlers, nodesFrom(() => nodes)],
  });
  return {
    app,
    codes,
    pair: (node: NodesNode) => {
      nodes = [home, node];
      live.clients[0]?.send({ event: "nodes.changed", data: { nodeId: node.id } });
    },
  };
}

test("Add a Node shows a fresh Join Code with its countdown, Copy and the three ways", async () => {
  const { app } = addNode();
  await userEvent.click(page.getByRole("button", { name: "Add a Node" }));
  const code = dialog().getByRole("textbox", { name: "Join Code" });
  await expect.element(code).toHaveValue(codeText(1));
  await expect
    .element(code)
    .toHaveAccessibleDescription(/^Expires in 59:\d\d\. It pairs one Node\.$/u);
  await expect.element(button("Copy Join Code")).toBeVisible();
  const ways = dialog().getByRole("list", { name: "Ways to use the code" });
  await expect.element(ways.getByText("First-run page.")).toBeVisible();
  await expect.element(ways.getByText(`slipstream join ${codeText(1)}`)).toBeVisible();
  await expect.element(ways.getByText(`SLIPSTREAM_JOIN_CODE=${codeText(1)}`)).toBeVisible();
  await expect.element(dialog().getByText("Waiting for the new Node to pair…")).toBeVisible();
  expect(app.router.state.location.search).toEqual({ add: "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d" });
});

test("a reload keeps the dialog on its code and says the code's text shows only once", async () => {
  const codes = joinCodeServer();
  const { id } = codes.mint();
  addNode({ path: `/nodes?add=${id}`, codes });
  await expect
    .element(
      dialog().getByText(
        /^A Join Code shows only when it is made\. This one still works for 59:\d\d\.$/u,
      ),
    )
    .toBeVisible();
  await expect
    .element(dialog().getByRole("textbox", { name: "Join Code" }))
    .not.toBeInTheDocument();
  const open = dialog().getByRole("list", { name: "Open Join Codes" });
  await expect.element(open.getByText("This code")).toBeVisible();
});

test("a pasted Node address has home dial the new Node, then says when it can't reach it", async () => {
  const codes = joinCodeServer();
  const { id } = codes.mint();
  addNode({ path: `/nodes?add=${id}`, codes });
  await userEvent.click(button("The new Node can't reach home?"));
  const address = dialog().getByRole("textbox", { name: "Node address" });
  await userEvent.fill(address, "not an address");
  await userEvent.click(button("home dials it"));
  await expect
    .element(address)
    .toHaveAccessibleDescription("The Node address is mistyped, cut short or not a Node address");

  await userEvent.fill(address, "SLN1-7Q2KD-M4ZP1");
  await userEvent.click(button("home dials it"));
  await expect
    .element(dialog().getByText("home is dialing the new Node at 203.0.113.9:8485…"))
    .toBeVisible();
  expect(codes.dialed).toEqual(["SLN1-7Q2KD-M4ZP1"]);

  codes.unreachable(id);
  await expect
    .element(dialog().getByRole("alert"))
    .toHaveTextContent(
      "home can't reach the new Node at 203.0.113.9:8485. Check that the port is open.",
    );
});

test("once the code is used the dialog shows the new Node's key and who dials, then its Roles pre-ticked through the Review", async () => {
  const codes = joinCodeServer();
  const { id } = codes.mint();
  const roles = rolesServer({ plans: [outcome({})], sets: [outcome({})] });
  const { app, pair } = addNode({
    path: `/nodes?add=${id}`,
    codes,
    handlers: [boardIs(board(homeSeat, vpsSeat)), ...roles.handlers],
  });
  await synced(app);
  codes.use(id, vps);
  pair(vps);
  await expect.element(dialog().getByText("vps paired.")).toBeVisible();
  await expect.element(dialog().getByText("SHA256:vps-key")).toBeVisible();
  await expect.element(dialog().getByText("home dials vps")).toBeVisible();

  await userEvent.click(button("Choose its Roles"));
  await expect
    .element(dialog().getByRole("checkbox", { name: "Searching", exact: true }))
    .toBeChecked();
  await expect
    .element(dialog().getByRole("checkbox", { name: "Downloading", exact: true }))
    .toBeChecked();
  await userEvent.click(button("Review"));
  const review = page.getByRole("dialog", { name: "Review the change" });
  await userEvent.click(review.getByRole("button", { name: "Apply" }));
  await expect.poll(() => app.router.state.location.pathname).toBe(`/nodes/${vpsId}/roles`);
  expect(roles.applied).toEqual([
    { basis: "basis-1", set: [{ nodeId: vpsId, roles: ["searching", "downloading"] }] },
  ]);
});

test("the new Node's Roles tick as its suggestions arrive", async () => {
  const codes = joinCodeServer();
  const { id } = codes.mint();
  const { app, pair } = addNode({
    path: `/nodes?add=${id}`,
    codes,
    handlers: [boardIs(board(homeSeat, { ...vpsSeat, suggested: ["searching"] }))],
  });
  await synced(app);
  codes.use(id, vps);
  pair(vps);
  await userEvent.click(button("Choose its Roles"));
  const downloading = dialog().getByRole("checkbox", { name: "Downloading", exact: true });
  await expect.element(downloading).not.toBeChecked();

  await boardBecomes(app.queryClient, board(homeSeat, vpsSeat));
  await expect.element(downloading).toBeChecked();
});

test("an expired code and a revoked one say so and offer a new code", async () => {
  const codes = joinCodeServer({ lifetimeMs: 1500 });
  const { app } = addNode({ codes });
  await userEvent.click(page.getByRole("button", { name: "Add a Node" }));
  await expect
    .element(dialog().getByText("This Join Code expired before a Node used it."))
    .toBeVisible();

  await userEvent.click(button("New Join Code"));
  await expect
    .element(dialog().getByRole("textbox", { name: "Join Code" }))
    .toHaveValue(codeText(2));
  expect(app.router.state.location.search).toEqual({ add: "2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e" });
});

test("the open Join Codes revoke one handed out by mistake, and revoking this one says so", async () => {
  const codes = joinCodeServer();
  codes.mint();
  const { id } = codes.mint();
  addNode({ path: `/nodes?add=${id}`, codes });
  const open = dialog().getByRole("list", { name: "Open Join Codes" });
  await expect.element(open.getByRole("listitem")).toHaveLength(2);
  await userEvent.click(open.getByRole("button", { name: /^Revoke the code made /u }));
  await expect.element(open.getByRole("listitem")).toHaveLength(1);

  await userEvent.click(open.getByRole("button", { name: "Revoke this code" }));
  await expect
    .element(dialog().getByText("This Join Code was revoked, so no Node can use it."))
    .toBeVisible();
});

test("with no link address, Add a Node says what to set", async () => {
  addNode({
    handlers: [
      http.post("*/api/v1/join-codes", () =>
        problem(409, "no_link_address", "The link has no address to put in a Join Code"),
      ),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Add a Node" }));
  await expect
    .element(dialog().getByRole("alert"))
    .toHaveTextContent(
      "home has no link address to put in a Join Code. Set link.advertise in its config.yaml, then restart it.",
    );
});
