import { type Request } from "@playwright/test";

import { expect, test } from "./fixtures.ts";

const isSessionRead = (request: Request) =>
  request.method() === "GET" && new URL(request.url()).pathname === "/api/v1/session";

test("signing out in one tab sends every tab of the session to sign-in", async ({
  adminPage,
  context,
  world,
}) => {
  const tab2 = await context.newPage();
  await tab2.goto("/");
  await expect(adminPage.getByText("Live updates: on")).toBeVisible();
  await expect(tab2.getByText("Live updates: on")).toBeVisible();

  await tab2.getByRole("button", { name: "Sign out" }).click();
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
  await expect(tab.getByText("Live updates: on")).toBeVisible();
  // The guard reads the session, and the stream's opening resync refetches it.
  await expect.poll(() => sessionReads.length).toBe(2);

  const { email, pin, displayName } = world.admin();
  const elsewhere = await (await newContext()).newPage();
  await elsewhere.goto("/sign-in");
  await elsewhere.getByLabel("Email").fill(email);
  await elsewhere.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially(pin);
  await expect(
    elsewhere.getByRole("heading", { name: `Signed in as ${displayName}` }),
  ).toBeVisible();
  await expect.poll(() => sessionReads.length).toBe(3);
});
