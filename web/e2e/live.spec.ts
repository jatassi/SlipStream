import { signIn } from "../harness/sign-in.ts";
import { type Request } from "@playwright/test";

import { expect, test } from "./fixtures.ts";
import { navigate, shellNav, signOut } from "./shell.ts";

const isSessionRead = (request: Request) =>
  request.method() === "GET" && new URL(request.url()).pathname === "/api/v1/session";

test("signing out in one tab sends every tab of the session to sign-in", async ({
  adminPage,
  context,
  world,
}) => {
  const tab2 = await context.newPage();
  await tab2.goto("/");
  await expect(shellNav(adminPage)).toBeVisible();
  await expect(shellNav(tab2)).toBeVisible();

  await signOut(tab2);
  await expect(adminPage).toHaveURL(`${world.baseURL}/sign-in?reason=signed-out`);
  await expect(adminPage.getByRole("heading", { name: "Sign in to SlipStream" })).toBeVisible();
});

test("a sign-in from another browser refreshes this tab's session once", async ({
  adminPage,
  newContext,
  world,
}) => {
  const tab = await adminPage.context().newPage();
  const sessionReads: Request[] = [];
  tab.on("requestfinished", (request) => {
    if (isSessionRead(request)) {
      sessionReads.push(request);
    }
  });
  await tab.goto("/");
  await expect(shellNav(tab)).toBeVisible();
  // The guard reads the session, and the stream's opening resync refetches it.
  await expect.poll(() => sessionReads.length).toBe(2);

  const elsewhere = await (await newContext()).newPage();
  await signIn(elsewhere, world.admin());
  await expect.poll(() => sessionReads.length).toBe(3);
});

test("live updates reconnect when the Node restarts on its address", async ({
  adminPage,
  world,
}) => {
  const reconnecting = adminPage.getByText("Reconnecting to SlipStream…");
  await expect(shellNav(adminPage)).toBeVisible();
  await expect(reconnecting).toHaveCount(0);

  await world.fw.node("kill", world.node.name);
  await expect(reconnecting).toBeVisible();

  const restarted = await world.fw.node("start", world.node.name);
  expect(restarted.listeners["http"]).toBe(world.node.listeners["http"]);
  await expect(reconnecting).toHaveCount(0);
  const { checks } = await world.fw.doctor();
  expect(checks.find((check) => check.n === 3)).toEqual({
    n: 3,
    name: "the cluster matches the Scenario",
    ok: true,
  });
});

test("a page not opened before fails inside the standing shell while the Node is down", async ({
  adminPage,
  world,
}) => {
  test.setTimeout(60_000);
  await expect(shellNav(adminPage)).toBeVisible();
  await world.fw.node("kill", world.node.name);
  await expect(adminPage.getByText("Reconnecting to SlipStream…")).toBeVisible();

  await navigate(adminPage, "Activity");
  // The page gives up after three refused requests. On Windows a refused
  // loopback connect takes about 4s, so the attempts and their 1s and 2s
  // backoffs take about 15s there.
  await expect(adminPage.getByRole("alert")).toContainText(
    "SlipStream isn't answering. Check that it's running, then try again.",
    { timeout: 20_000 },
  );
  await expect(shellNav(adminPage)).toBeVisible();

  await world.fw.node("start", world.node.name);
  await expect(adminPage.getByText("Reconnecting to SlipStream…")).toHaveCount(0);
  await adminPage.getByRole("button", { name: "Try again" }).click();
  await expect(adminPage.getByRole("heading", { name: "Activity", level: 1 })).toBeVisible();
});
