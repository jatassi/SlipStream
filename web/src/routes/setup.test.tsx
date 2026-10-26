import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { readPref, writePref } from "../shell/prefs";
import { renderApp } from "../test/render-app";
import {
  forgetKeptCode,
  joinExisting,
  rightCode,
  setupCode,
  startNew,
  typeCode,
  unclaimedNode,
} from "../test/setup-node";
import { staleLinkSentence } from "./-setup/use-setup-code";

forgetKeptCode();

test("an Unclaimed Node sends / to setup", async () => {
  const { router } = renderApp({ path: "/", handlers: unclaimedNode().handlers });
  await expect.element(page.getByRole("heading", { name: "Set up SlipStream" })).toBeVisible();
  expect(router.state.location.href).toBe("/setup");
});

test("the Setup code shows grouped as it is typed", async () => {
  renderApp({ path: "/setup", handlers: unclaimedNode().handlers });
  await userEvent.type(setupCode(), "7k2m q9xd 4hbt");
  await expect.element(setupCode()).toHaveValue("7K2M-Q9XD-4HBT");
});

test("a wrong Setup code fails on its own screen before any other field", async () => {
  const { node, handlers } = unclaimedNode();
  renderApp({ path: "/setup", handlers });
  await typeCode("0000 0000 0000");
  await expect.element(setupCode()).toHaveAccessibleDescription("The Setup code is wrong");
  await expect.element(page.getByLabelText("Email")).not.toBeInTheDocument();
  expect(node.headers).toEqual([{ path: "/api/v1/setup/code", code: "000000000000" }]);
});

test("a right Setup code leads to the choice, with Start a new SlipStream first", async () => {
  const { node, handlers } = unclaimedNode();
  renderApp({ path: "/setup", handlers });
  await typeCode("7k2m q9xd 4hbt");
  const choices = page.getByRole("list", { name: "What this Node becomes" }).getByRole("button");
  await expect.element(choices.nth(0)).toHaveAccessibleName("Start a new SlipStream");
  await expect.element(choices.nth(1)).toHaveAccessibleName("Join an existing SlipStream");
  expect(readPref("setupCode")).toBe(rightCode);
  expect(node.headers[0]).toEqual({ path: "/api/v1/setup/code", code: rightCode });
});

test("a setup link fills in and checks the code, and leaves the address bar", async () => {
  const { router } = renderApp({
    path: "/setup#code=7k2m%20q9xd%204hbt",
    handlers: unclaimedNode().handlers,
  });
  await expect.element(page.getByText("Setup code filled in")).toBeVisible();
  await expect.element(startNew()).toBeVisible();
  expect(router.state.location.href).toBe("/setup");
  expect(router.history.location.hash).toBe("");
});

test("a setup link from an earlier start of the Node says so", async () => {
  renderApp({ path: "/setup#code=AAAA-BBBB-CCCC", handlers: unclaimedNode().handlers });
  await expect.element(page.getByText(staleLinkSentence)).toBeVisible();
  await expect.element(setupCode()).toHaveValue("");
});

test("a second setup link in the same tab fills in", async () => {
  const { router } = renderApp({ path: "/setup", handlers: unclaimedNode().handlers });
  await expect.element(setupCode()).toBeVisible();
  await router.navigate({ to: "/setup", hash: `code=${rightCode}` });
  await expect.element(page.getByText("Setup code filled in")).toBeVisible();
  await expect.poll(() => router.history.location.hash).toBe("");
});

test("a reload keeps the code this tab already checked", async () => {
  writePref("setupCode", rightCode);
  renderApp({ path: "/setup", handlers: unclaimedNode().handlers });
  await expect.element(startNew()).toBeVisible();
  await expect.element(setupCode()).not.toBeInTheDocument();
});

test("a kept code the Node no longer takes says the Setup code changed", async () => {
  writePref("setupCode", "AAAABBBBCCCC");
  renderApp({ path: "/setup", handlers: unclaimedNode().handlers });
  await expect
    .element(
      page.getByText(
        "This Node restarted, so its Setup code changed. Read the new one from its log.",
      ),
    )
    .toBeVisible();
  expect(readPref("setupCode")).toBe("");
});

test("a removed Node leads with the Removed sentence and with Join", async () => {
  writePref("setupCode", rightCode);
  const removed = { controller: "home", at: "2026-10-17T09:30:00Z" };
  renderApp({ path: "/setup", handlers: unclaimedNode(removed).handlers });
  const day = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short" }).format(
    new Date(removed.at),
  );
  await expect
    .element(
      page.getByText(
        `This Node was removed from home on ${day}. Join with a new Join Code to use it again.`,
      ),
    )
    .toBeVisible();
  const choices = page.getByRole("list", { name: "What this Node becomes" }).getByRole("button");
  await expect.element(choices.nth(0)).toHaveAccessibleName("Join an existing SlipStream");
});

test("Back returns to the choice before anything is created", async () => {
  writePref("setupCode", rightCode);
  renderApp({ path: "/setup", handlers: unclaimedNode().handlers });
  await userEvent.click(startNew());
  await expect.element(page.getByRole("heading", { name: "Start a new SlipStream" })).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Back" }));
  await expect.element(joinExisting()).toBeVisible();
});
