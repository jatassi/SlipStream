import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, navigate, shellNav } from "./shell.ts";

test("the shell reaches every destination and titles each tab", async ({
  adminPage: page,
  world,
}) => {
  await expect(shellNav(page)).toBeVisible();
  await expect(page).toHaveTitle("Library · SlipStream");

  await navigate(page, "Activity");
  await expect(page.getByRole("heading", { name: "Activity" })).toBeVisible();
  await expect(page).toHaveTitle("Activity · SlipStream");
  await expectAccessible(page);

  await navigate(page, "Nodes");
  await expect(
    page.getByRole("heading", { name: isPhone(page) ? "Nodes" : world.node.name, level: 1 }),
  ).toBeVisible();
  await expect(page).toHaveTitle(`${world.node.name} · SlipStream`);

  await navigate(page, "Library");
  await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
  await expect(page).toHaveTitle("Library · SlipStream");
});

test("the sidebar collapses to a rail that stays, and the phone's More lists the rest", async ({
  adminPage: page,
}) => {
  if (isPhone(page)) {
    await page
      .getByRole("navigation", { name: "Tabs" })
      .getByRole("button", { name: "More" })
      .click();
    const sheet = page.getByRole("dialog", { name: "More" });
    await expect(sheet.getByRole("button", { name: "Nodes" })).toBeVisible();
    await expect(sheet.getByText(/^SlipStream \S+$/u)).toBeVisible();
    await expectAccessible(page);
    await sheet.getByRole("button", { name: "Cancel" }).click();
    await expect(sheet).toHaveCount(0);
    return;
  }
  const nav = page.getByRole("navigation", { name: "Main" });
  await page.getByRole("button", { name: "Collapse sidebar" }).click();
  await expect.poll(async () => (await nav.boundingBox())?.width).toBe(60);
  await expectAccessible(page);
  await page.reload();
  await expect(page.getByRole("button", { name: "Expand sidebar" })).toBeVisible();
  await expect.poll(async () => (await nav.boundingBox())?.width).toBe(60);
});
