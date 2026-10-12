import { expect, expectAccessible, test } from "./fixtures.ts";

test.use({ scenario: "first-run" });

test("a fresh Node takes a person from the Setup code to the first Admin", async ({
  page,
  world,
  browserName,
}) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Set up SlipStream" })).toBeVisible();
  expect(page.url()).toMatch(
    browserName === "chromium"
      ? /^http:\/\/home\.localhost:\d+\/setup$/u
      : /^http:\/\/127\.0\.0\.\d+:\d+\/setup$/u,
  );
  await expectAccessible(page);

  const setupCode = page.getByLabel("Setup code");
  await setupCode.pressSequentially("0000 0000 0000");
  await page.getByLabel("Email").fill("grace@example.test");
  await page.getByLabel("Display name").fill("Grace Hopper");
  await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially("1357");
  await page.getByRole("textbox", { name: "PIN again", exact: true }).pressSequentially("1357");
  await page.getByRole("button", { name: "Create Admin" }).click();
  await expect(setupCode).toHaveAccessibleDescription("The Setup code is wrong");

  await setupCode.clear();
  await setupCode.pressSequentially((await world.setupCode()).toLowerCase().replaceAll("-", " "));
  await page.getByRole("button", { name: "Create Admin" }).click();
  await expect(page.getByLabel("API key")).toHaveValue(/^ssk_[A-Za-z0-9_-]{43}$/u);
  await expectAccessible(page);

  await page.getByRole("button", { name: "Continue" }).click();
  const signedIn = page.getByRole("heading", { name: "Signed in as Grace Hopper" });
  await expect(signedIn).toBeVisible();
  await page.reload();
  await expect(signedIn).toBeVisible();
  await page.goto("/setup");
  await expect(signedIn).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/`);
  expect(await world.api("GET", "/api/v1/status")).toEqual({
    status: 200,
    body: { claim: "controller" },
  });
});
