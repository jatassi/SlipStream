import type { Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";

test.use({ scenario: "series" });
test.describe.configure({ timeout: 120_000 });

const card = (page: Page) => page.getByRole("article", { name: "TVDB" });
const toasts = (page: Page) => page.getByRole("region", { name: "Notifications" });

async function setKey(page: Page, key: string, pin: string) {
  await card(page)
    .getByRole("button", { name: /^(Set|Change) key$/u })
    .click();
  const dialog = page.getByRole("dialog", { name: "Set your TVDB key" });
  await dialog.getByRole("textbox", { name: "Key" }).fill(key);
  await dialog.getByRole("textbox", { name: "PIN" }).fill(pin);
  await dialog.getByRole("button", { name: "Test and save" }).click();
  return dialog;
}

test("a TVDB key and PIN are tested before they're kept, and TVDB's refusal names the key or the PIN", async ({
  adminPage: page,
  world,
}) => {
  await page.goto("/settings/metadata");
  await expect(card(page)).toContainText("TVDB is working.");
  await expect(card(page)).toContainText("Metadata provided by TheTVDB.");

  const kept = await setKey(page, "e2e-tvdb-key", "1234");
  await expect(kept).toBeHidden();
  await expect(toasts(page)).toContainText("SlipStream now calls TVDB with your key and PIN.");
  await expect(card(page)).toContainText(/Your key, set /u);

  await world.fw.cue("tvdb-reject-key", { key: "e2e-refused-key" });
  const refusedKey = await setKey(page, "e2e-refused-key", "");
  await expect(refusedKey.getByRole("textbox", { name: "Key" })).toHaveAccessibleDescription(
    "The metadata provider refused that key, so SlipStream didn't save it",
  );
  await refusedKey.getByRole("button", { name: "Cancel" }).click();

  await world.fw.cue("tvdb-reject-pin", {});
  const refusedPin = await setKey(page, "e2e-tvdb-key", "9999");
  await expect(refusedPin.getByRole("textbox", { name: "PIN" })).toHaveAccessibleDescription(
    "The metadata provider refused that PIN, so SlipStream didn't save it",
  );
  await refusedPin.getByRole("button", { name: "Cancel" }).click();

  await card(page).getByRole("button", { name: "Test" }).click();
  await expect(toasts(page)).toContainText(
    "TVDB refused your PIN. Check it and save it again with your key, or remove your key.",
  );
});

test("an expired TVDB token logs in again, and the search goes through", async ({
  adminPage: page,
  world,
}) => {
  await page.goto("/add?contentType=series&query=lantern");
  await expect(page.getByRole("button", { name: "Add The Lantern Keepers" })).toBeVisible();

  await world.fw.fault.set({
    id: "logins",
    target: "tvdb",
    route: "login",
    fault: "slow-body=1GiB/s",
  });
  await world.fw.cue("tvdb-expire-tokens", {});
  await page.goto("/add?contentType=series&query=harbor");
  await expect(page.getByRole("button", { name: "Add Harbor Nights" })).toBeVisible();
  await world.fw.fault.wait({ rule: "logins", timeout: 10_000 });
});
