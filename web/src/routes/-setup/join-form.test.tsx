import { HttpResponse } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { readPref, writePref } from "../../shell/prefs";
import { problem } from "../../test/handlers";
import { renderApp } from "../../test/render-app";
import {
  awaiting,
  forgetKeptCode,
  join,
  joining,
  rightCode,
  setupCode,
  unclaimedNode,
  wrongCode,
} from "../../test/setup-node";

forgetKeptCode();

test("a Join that pairs at once lands on the Worker page", async () => {
  writePref("setupCode", rightCode);
  const server = joining((node) => {
    node.claim = "worker";
    return HttpResponse.json({ state: "joined", nodeId: "1" }, { status: 201 });
  });
  const { router } = renderApp({ path: "/setup", handlers: server.handlers });
  await join("  SLP1-AAAA  ");
  await expect.poll(() => router.state.location.pathname).toBe("/worker");
  expect(server.bodies).toEqual([{ code: "SLP1-AAAA" }]);
  expect(readPref("setupCode")).toBe("");
});

test("a Join that waits for the dial shows the Node address and moves on once joined", async () => {
  writePref("setupCode", rightCode);
  const server = joining((node) => {
    node.join = awaiting;
    return HttpResponse.json(awaiting, { status: 202 });
  });
  const { router } = renderApp({ path: "/setup", handlers: server.handlers });
  await join();
  await expect.element(page.getByLabelText("Node address")).toHaveValue("SLN1-8F2KQ9XD4HBT");
  await expect
    .element(page.getByText("Paste this Node address into Add a Node on your SlipStream."))
    .toBeVisible();
  await expect
    .element(page.getByLabelText("Node address"))
    .toHaveAccessibleDescription(/^Expires in (10:00|9:\d\d)\.$/u);

  server.node.join = { state: "joined", nodeId: "1", nodeAddress: "SLN1-8F2KQ9XD4HBT" };
  server.node.claim = "worker";
  await expect.poll(() => router.state.location.pathname, { timeout: 5000 }).toBe("/worker");
});

test("a reload while waiting for the dial resumes the wait", async () => {
  writePref("setupCode", rightCode);
  const server = unclaimedNode();
  server.node.join = awaiting;
  renderApp({ path: "/setup", handlers: server.handlers });
  await expect.element(page.getByLabelText("Node address")).toHaveValue("SLN1-8F2KQ9XD4HBT");
});

const refusals = [
  [
    422,
    "join_code_damaged",
    "That Join Code is mistyped or cut short. Copy it again from Add a Node on your SlipStream.",
  ],
  [
    403,
    "join_code_unknown",
    "Your SlipStream has no such Join Code. Make a new one in Add a Node.",
  ],
  [403, "join_code_expired", "That Join Code has expired. Make a new one in Add a Node."],
  [403, "join_code_used", "That Join Code was already used. Make a new one in Add a Node."],
  [
    502,
    "controller_key_mismatch",
    "The SlipStream that responded doesn't hold the key in this Join Code. Make a new Join Code on the SlipStream you meant to join.",
  ],
  [
    409,
    "node_unaddressed",
    "This Node can't reach your SlipStream and has no network address your SlipStream could dial. Connect both to the same network, then try again.",
  ],
  [
    409,
    "restart_required",
    "This Node can join only after a restart. Restart it, then enter the Join Code again.",
  ],
  [
    409,
    "setup_busy",
    "Another setup is running on this Node. Wait for it to finish, then try again.",
  ],
  [409, "already_claimed", "Someone has already set up this Node."],
  [409, "join_pending", "This Node is already waiting to join a SlipStream."],
  [500, "internal", "Joining failed on this Node. Try again, and check its log if it fails again."],
] as const;

test.each(refusals)(
  "a Join refused with %i %s says so in plain words",
  async (status, code, sentence) => {
    writePref("setupCode", rightCode);
    const server = joining(() => problem(status, code, "The title the page doesn't show"));
    renderApp({ path: "/setup", handlers: server.handlers });
    await join();
    await expect.element(page.getByText(sentence)).toBeVisible();
  },
);

test("a Join refused for the Setup code goes back to the code screen", async () => {
  writePref("setupCode", rightCode);
  const server = joining(() => wrongCode());
  renderApp({ path: "/setup", handlers: server.handlers });
  await join();
  await expect
    .element(
      page.getByText(
        "This Node restarted, so its Setup code changed. Read the new one from its log.",
      ),
    )
    .toBeVisible();
  await expect.element(setupCode()).toBeVisible();
});

const failures = [
  [
    "controller_unreachable",
    "Your SlipStream didn't dial this Node before the address expired. Try again with a new Join Code.",
  ],
  ["join_code_unknown", "Your SlipStream has no such Join Code. Make a new one in Add a Node."],
  ["join_code_expired", "That Join Code has expired. Make a new one in Add a Node."],
  ["join_code_used", "That Join Code was already used. Make a new one in Add a Node."],
  [
    "controller_key_mismatch",
    "The SlipStream that responded doesn't hold the key in this Join Code. Make a new Join Code on the SlipStream you meant to join.",
  ],
  ["internal", "Joining failed on this Node. Try again, and check its log if it fails again."],
] as const;

test.each(failures)(
  "a Join that failed while waiting with %s says so and offers another code",
  async (code, sentence) => {
    writePref("setupCode", rightCode);
    const server = unclaimedNode();
    server.node.join = { ...awaiting, state: "failed", problem: { code, detail: "dial refused" } };
    renderApp({ path: "/setup", handlers: server.handlers });
    await expect.element(page.getByText(sentence)).toBeVisible();
    await userEvent.click(page.getByRole("button", { name: "Try another Join Code" }));
    await expect.element(page.getByLabelText("Join Code")).toHaveValue("");
  },
);
