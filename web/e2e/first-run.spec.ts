import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

test.use({ scenario: "first-run" });

const claimed = {
  status: 200,
  body: { claim: "controller", passkeys: { available: false, reason: "ip_address" } },
};

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
  await expect(page.getByRole("textbox", { name: "API key" })).toHaveValue(
    /^ssk_[A-Za-z0-9_-]{43}$/u,
  );
  await expectAccessible(page);

  await page.getByRole("button", { name: "Continue" }).click();
  const signedIn = page.getByRole("heading", { name: "Library" });
  await expect(signedIn).toBeVisible();
  if (isPhone(page)) {
    await page
      .getByRole("navigation", { name: "Tabs" })
      .getByRole("button", { name: "More" })
      .click();
  }
  await expect(page.getByText("Grace Hopper")).toBeVisible();
  await page.reload();
  await expect(signedIn).toBeVisible();
  await page.goto("/setup");
  await expect(signedIn).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/library`);
  expect(await world.api("GET", "/api/v1/status")).toEqual(claimed);
});

test("a setup link fills in the Setup code and drops it from the address bar", async ({
  page,
  world,
}) => {
  const code = await world.setupCode();
  await page.goto(`${world.baseURL}/setup#code=${code}`);
  const setupCode = page.getByLabel("Setup code");
  await expect(setupCode).toHaveValue(code);
  await expect(setupCode).toHaveAccessibleDescription("Setup code filled in");
  await expect.poll(() => page.url()).toBe(`${world.baseURL}/setup`);
  await expectAccessible(page);

  await page.getByLabel("Email").fill("grace@example.test");
  await page.getByLabel("Display name").fill("Grace Hopper");
  await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially("1357");
  await page.getByRole("textbox", { name: "PIN again", exact: true }).pressSequentially("1357");
  await page.getByRole("button", { name: "Create Admin" }).click();
  await expect(page.getByRole("textbox", { name: "API key" })).toHaveValue(
    /^ssk_[A-Za-z0-9_-]{43}$/u,
  );
  expect(await world.api("GET", "/api/v1/status")).toEqual(claimed);
});
