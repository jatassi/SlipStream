import { expect, expectAccessible, test } from "./fixtures.ts";
import { frame } from "./manual-search.ts";
import { cinder, scanned, searchNow, wantEnglish } from "./subtitles.ts";

test.use({ scenario: "subtitles" });
test.describe.configure({ timeout: 300_000 });

test("a refused SubDL key isn't saved, another is, and once SubDL revokes it Settings says to replace it", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await world.fw.cue("subdl-revoke-key", { key: "not-a-key" });
  await page.goto("/settings/subtitles");
  const providers = page.getByRole("list", { name: "Subtitle Providers" });
  await providers.getByRole("button", { name: "Add key" }).first().click();
  const dialog = page.getByRole("dialog", { name: "Add your SubDL key" });
  await expect(dialog).toContainText(/SubDL password/u);

  await dialog.getByRole("textbox", { name: "Key" }).fill("not-a-key");
  await dialog.getByRole("button", { name: "Check key" }).click();
  await expect(dialog).toContainText("Not saved");
  await expect(dialog).toContainText("Not Authorized");
  await expectAccessible(page);
  await frame(page, testInfo, "refused");

  await dialog.getByRole("button", { name: "Try another key" }).click();
  await dialog.getByRole("textbox", { name: "Key" }).fill("e2e-subdl");
  await dialog.getByRole("button", { name: "Check key" }).click();
  await expect(dialog).toContainText(
    "The key works. It belongs to slipstream on SubDL's Free plan",
  );
  await frame(page, testInfo, "accepted");
  await dialog.getByRole("button", { name: "Done" }).click();
  await expect(providers.getByRole("button", { name: "Replace key" })).toBeVisible();

  await wantEnglish(world, "Movies");
  await scanned(world, cinder);
  await world.fw.cue("subdl-revoke-key", {});
  await searchNow(world, cinder);
  await expect(providers).toContainText(
    /SubDL rejected the key at .+ and responded "Not Authorized"\. Replace the key\./u,
    { timeout: 60_000 },
  );
  await expectAccessible(page);
  await frame(page, testInfo, "revoked");
});
