import type { Page } from "@playwright/test";

import { expect, expectAccessible, test } from "./fixtures.ts";
import { navigate } from "./shell.ts";

test.use({ scenario: "home" });

async function backUpNow(page: Page): Promise<void> {
  await page.getByRole("button", { name: "Back up now" }).click();
  await expect(page.getByText("Backing up…").first()).toBeVisible();
  const manual = page
    .getByRole("list", { name: "Backups" })
    .getByRole("listitem")
    .filter({ hasText: "Manual" });
  await expect(manual).toContainText("Checked", { timeout: 60_000 });
}

test("Back up now takes a Backup, and the list shows it checked", async ({ adminPage: page }) => {
  await page.goto("/settings/backups");
  await expect(page.getByRole("heading", { name: "Backup passphrase" })).toBeVisible();
  await backUpNow(page);
  await expectAccessible(page);
});

test("on a phone Backups opens from Settings' System list, and Back up now takes one @phone", async ({
  adminPage: page,
}) => {
  await navigate(page, "Settings");
  await page.getByRole("list", { name: "System" }).getByRole("link", { name: "Backups" }).click();
  await expect(page).toHaveURL(/\/settings\/backups$/u);
  await backUpNow(page);
  await expectAccessible(page);
});
