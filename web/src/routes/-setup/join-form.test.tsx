import { HttpResponse, http } from "msw";
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

test("a Node claimed elsewhere while this tab waits for the dial forgets the Setup code", async () => {
  writePref("setupCode", rightCode);
  const server = unclaimedNode();
  server.node.join = awaiting;
  let claimed = false;
  const { router } = renderApp({
    path: "/setup",
    handlers: [
      http.get("*/api/v1/setup/worker", () =>
        claimed ? problem(409, "already_claimed", "This Node is already set up") : undefined,
      ),
      ...server.handlers,
    ],
  });
  await expect.element(page.getByLabelText("Node address")).toHaveValue("SLN1-8F2KQ9XD4HBT");
  claimed = true;
  server.node.claim = "worker";
  await expect.poll(() => router.state.location.pathname, { timeout: 5000 }).not.toBe("/setup");
  expect(readPref("setupCode")).toBe("");
});

const refusals = [
  [403, "join_code_expired", "That Join Code has expired. Make a new one in Add a Node."],
  [409, "already_claimed", "Someone has already set up this Node."],
] as const;

test.each(refusals)(
  "a Join refused with %i %s says so in plain words",
  async (status, code, sentence) => {
    writePref("setupCode", rightCode);
    const server = joining(() => problem(status, code, "The title the page doesn't show"));
    renderApp({ path: "/setup", handlers: server.handlers });
    await join();
    await expect.element(page.getByText(sentence)).toBeVisible();
    expect(readPref("setupCode")).toBe(code === "already_claimed" ? "" : rightCode);
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
    "Your SlipStream didn't connect to this Node before its Node address expired. Try again with a new Join Code.",
  ],
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
