import { z } from "zod";

import { expect, expectAccessible, test } from "./fixtures.ts";
import { openSettingsSection } from "./shell.ts";
import { screenshotWholePage } from "./whole-page.ts";

const Thresholds = z.object({
  thresholds: z.object({ unregisteredGap: z.object({ value: z.number() }) }),
  cards: z.array(z.object({ nodeId: z.string() })),
});
const Limits = z.object({ activeDownloads: z.object({ value: z.number() }) });
const cardGoes = "This card is removed once this Node stops downloading.";

test("Download Clients shows home's built-in client, saves a threshold, and its Torrent client saves a limit", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await openSettingsSection(page, "Download Clients");
  await expect(page).toHaveURL(/\/settings\/download-clients$/u);

  const card = page.getByRole("article", { name: "Built-in client on home" });
  await expect(
    card.getByText(
      /^Listening on 127\.0\.0\.\d+:(\d+)\. No peer from the internet has connected to port \1 yet, so it may not be reachable\.$/u,
    ),
  ).toBeVisible();
  await expect(card.getByText(/^0 torrents, 0 bytes uploaded/u)).toBeVisible();
  await card.getByRole("button", { name: "Details" }).click();
  await expect(card.getByText(/^-LT\w{4}-$/u)).toBeVisible();
  await expect(card.getByText(/^libtorrent\/\d+\.\d+\.\d+/u)).toBeVisible();
  await expect(card.getByText(cardGoes, { exact: true })).toBeVisible();
  await expect(card.getByRole("button", { name: /remove/iu })).toHaveCount(0);

  await page.getByRole("textbox", { name: "Unregistered twice, this far apart" }).fill("45");
  await page.getByRole("main").getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Download Clients settings saved",
  );
  const saved = Thresholds.parse((await world.api("GET", "/api/v1/download-clients")).body);
  expect(saved.thresholds.unregisteredGap.value, "45 minutes in seconds").toBe(2700);
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("download-clients.png"));

  await card.getByRole("link", { name: "Torrent client settings on home" }).click();
  await expect(page).toHaveURL(/\/nodes\/[\da-f-]+\/torrent-client$/u);
  await page.getByRole("textbox", { name: "Downloads at once" }).fill("1");
  await page.getByRole("main").getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Torrent client settings saved",
  );
  const [home] = saved.cards;
  if (home === undefined) {
    throw new Error("home has no Download Clients card");
  }
  const limits = await world.api("GET", `/api/v1/download-clients/${home.nodeId}/settings`);
  expect(Limits.parse(limits.body).activeDownloads.value).toBe(1);
  await screenshotWholePage(page, testInfo.outputPath("torrent-client.png"));
});

test.describe("an external client", () => {
  test.use({ scenario: "external-clients" });
  test.describe.configure({ timeout: 240_000 });

  test("an unreachable Transmission leads Activity > Torrents with a red card, and Check now clears it once it's back", async ({
    adminPage: page,
    world,
  }, testInfo) => {
    await world.fw.fault.set({
      id: "transmission-down",
      target: "transmission",
      node: "home",
      fault: "reset",
    });
    await page.goto("/activity/torrents");
    const card = page.getByRole("region", { name: "Transmission on home" });
    await expect(card.getByText("Not responding", { exact: true })).toBeVisible({
      timeout: 180_000,
    });
    await expect(
      card.getByText(
        /^Transmission isn't responding\. Check that it's running and that home can reach .+\.$/u,
      ),
    ).toBeVisible();
    await expectAccessible(page);
    await screenshotWholePage(page, testInfo.outputPath("client-away.png"));

    await world.fw.fault.clear("transmission-down");
    await card.getByRole("button", { name: "Check now" }).click();
    await expect(card).toHaveCount(0);
  });
});
