import { z } from "zod";

import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

test.use({ scenario: "home" });

const Nodes = z.object({ nodes: z.array(z.object({ id: z.string(), name: z.string() })) });
const Settings = z.object({ sendLimit: z.object({ value: z.number() }) });

test("seedbox's Transfers say nothing moves and save a send limit the Node reads back", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const { nodes } = Nodes.parse((await world.api("GET", "/api/v1/nodes")).body);
  const seedbox = nodes.find(({ name }) => name === "seedbox");
  if (seedbox === undefined) {
    throw new Error("the home Scenario has no seedbox");
  }
  await page.goto(`/nodes/${seedbox.id}`);
  const sections = isPhone(page)
    ? page.getByRole("list", { name: "seedbox" })
    : page.getByRole("navigation", { name: "seedbox" });
  await sections.getByRole("link", { name: "Transfers" }).click();
  await expect(page).toHaveURL(/\/nodes\/[\da-f-]+\/transfers$/u);
  await expect(page.getByText("Nothing moving", { exact: true })).toBeVisible();

  await page.getByRole("textbox", { name: "Send limit" }).fill("10");
  await page.getByRole("main").getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Transfer settings saved",
  );
  const saved = await world.api("GET", `/api/v1/transfers/nodes/${seedbox.id}/settings`);
  expect(Settings.parse(saved.body).sendLimit.value, "10 MB/s in bytes").toBe(10 * 1024 * 1024);
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("transfers.png"), fullPage: true });
});
