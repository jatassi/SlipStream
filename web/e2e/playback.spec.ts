import { expect, expectAccessible, test } from "./fixtures.ts";
import { frame } from "./manual-search.ts";
import { openServer, openTab } from "./media-servers.ts";

test.use({ scenario: "playback" });

const server = "Living room Jellyfin";
const player = "Bedroom TV";
const suggested = "1080p HDR10";

test("Bedroom TV's suggestion on the Playback report starts a New Target that, once created, is listed in Targets", async ({
  adminPage: page,
}, testInfo) => {
  await openServer(page, server);
  await openTab(page, server, "Playback report");
  const row = page
    .getByRole("list", { name: `Players on ${server}` })
    .getByRole("listitem")
    .filter({ hasText: player });
  await expect(row).toContainText(suggested);
  await expectAccessible(page);
  await frame(page, testInfo, "suggest");

  await row.getByRole("link", { name: "Add this Target" }).click();
  await expect(page.getByRole("heading", { name: "New Target", level: 1 })).toBeVisible();
  const main = page.getByRole("main");
  await expect(main).toContainText(player);
  await expect(main).toContainText(server);
  await expect(page.getByRole("textbox", { name: "Name" })).toHaveValue(suggested);
  await expectAccessible(page);
  await frame(page, testInfo, "new-target");

  await page.getByRole("button", { name: "Create Target" }).click();
  await expect(page).toHaveURL(/\/settings\/targets$/u);
  await expect(
    page
      .getByRole("table", { name: "Targets" })
      .getByRole("link", { name: suggested, exact: true }),
  ).toBeVisible();
  await expectAccessible(page);
  await frame(page, testInfo, "targets");
});
