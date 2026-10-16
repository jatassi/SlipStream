import { expect, test } from "./fixtures.ts";
import { navigate } from "./shell.ts";

test.describe.configure({ timeout: 120_000 });

test("Restart from the account menu shows the restarting state and lands back on the same page", async ({
  adminPage: page,
}) => {
  await navigate(page, "Nodes");
  await page.getByRole("link", { name: "Web and API" }).click();
  await expect(page.getByRole("textbox", { name: "Port", exact: true })).toBeVisible();
  const before = page.url();

  await page.getByRole("button", { name: "Account" }).click();
  await page.getByRole("menuitem", { name: "Restart SlipStream…" }).click();
  const reloaded = page.waitForEvent("load", { timeout: 90_000 });
  await page
    .getByRole("dialog", { name: "Restart SlipStream?" })
    .getByRole("button", { name: "Restart" })
    .click();
  await expect(page.getByText("Restarting SlipStream…")).toBeVisible();

  await reloaded;
  expect(page.url()).toBe(before);
  await expect(page.getByRole("textbox", { name: "Port", exact: true })).toBeVisible();
  await expect(page.getByText("Restarting SlipStream…")).toHaveCount(0);
});
