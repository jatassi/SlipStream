import { z } from "zod";

import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, navigate } from "./shell.ts";

const Thresholds = z.object({
  thresholds: z.object({ unregisteredGap: z.object({ value: z.number() }) }),
  cards: z.array(z.object({ nodeId: z.string() })),
});
const Limits = z.object({ activeDownloads: z.object({ value: z.number() }) });

test("Download Clients shows home's built-in client, saves a threshold, and its Torrent client saves a limit", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await navigate(page, "Settings");
  const sections = isPhone(page)
    ? page.getByRole("list", { name: "Settings" })
    : page.getByRole("navigation", { name: "Settings" });
  await sections.getByRole("link", { name: "Download Clients" }).click();
  await expect(page).toHaveURL(/\/settings\/download-clients$/u);

  const card = page.getByRole("article", { name: "Built-in client on home" });
  await expect(
    card.getByText(
      /^Listening on 127\.0\.0\.\d+:(\d+)\. No peer from the internet has connected in yet, so port \1 is unproven\.$/u,
    ),
  ).toBeVisible();
  await expect(card.getByText(/^0 torrents, 0 bytes uploaded/u)).toBeVisible();
  await card.getByRole("button", { name: "Details" }).click();
  await expect(card.getByText(/^-LT\w{4}-$/u)).toBeVisible();
  await expect(card.getByText(/^libtorrent\/\d+\.\d+\.\d+/u)).toBeVisible();
  await expect(card.getByText("Goes away when this Node stops downloading.")).toBeVisible();
  await expect(card.getByRole("button", { name: /remove/iu })).toHaveCount(0);

  await page.getByRole("textbox", { name: "Unregistered twice, this far apart" }).fill("45");
  await page.getByRole("main").getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Download Clients settings saved",
  );
  const saved = Thresholds.parse((await world.api("GET", "/api/v1/download-clients")).body);
  expect(saved.thresholds.unregisteredGap.value, "45 minutes in seconds").toBe(2700);
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("download-clients.png"), fullPage: true });

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
  await page.screenshot({ path: testInfo.outputPath("torrent-client.png"), fullPage: true });
});
