import { HttpResponse } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { readPref, writePref } from "../../shell/prefs";
import { problem } from "../../test/handlers";
import { renderApp } from "../../test/render-app";
import {
  claiming,
  fillStart,
  forgetKeptCode,
  rightCode,
  signedIn,
  startNew,
} from "../../test/setup-node";

forgetKeptCode();

test("Start a new SlipStream claims with the checked code and leaves the page", async () => {
  writePref("setupCode", rightCode);
  const server = claiming(() => HttpResponse.json(signedIn, { status: 201 }));
  const { router } = renderApp({ path: "/setup", handlers: server.handlers });
  await userEvent.click(startNew());
  await fillStart("1234");
  await expect.poll(() => router.state.location.pathname).not.toBe("/setup");
  expect(server.claims).toEqual([
    {
      code: rightCode,
      body: { email: "ada@example.test", displayName: "Ada Okafor", pin: "1234" },
    },
  ]);
  expect(readPref("setupCode")).toBe("");
});

test("mismatched PINs never reach the server", async () => {
  writePref("setupCode", rightCode);
  const server = claiming(() => HttpResponse.json(signedIn, { status: 201 }));
  renderApp({ path: "/setup", handlers: server.handlers });
  await userEvent.click(startNew());
  await fillStart("4321");
  await expect.element(page.getByText("The PINs don't match")).toBeVisible();
  expect(server.claims).toEqual([]);
});

test("a lost race to claim says someone set the Node up and offers Sign in", async () => {
  writePref("setupCode", rightCode);
  const server = claiming(() => problem(409, "already_claimed", "This Node is already set up"));
  renderApp({ path: "/setup", handlers: server.handlers });
  await userEvent.click(startNew());
  await fillStart("1234");
  await expect.element(page.getByText("Someone has already set up this Node.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Sign in" })).toBeVisible();
  expect(readPref("setupCode")).toBe("");
});

test("Start a new SlipStream is refused while a Join waits, with a way back to the wait", async () => {
  writePref("setupCode", rightCode);
  const server = claiming(() =>
    problem(409, "join_pending", "This Node is waiting to join a SlipStream"),
  );
  renderApp({ path: "/setup", handlers: server.handlers });
  await userEvent.click(startNew());
  await fillStart("1234");
  await expect.element(page.getByText("This Node is waiting to join a SlipStream.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Back to the wait" })).toBeVisible();
});
