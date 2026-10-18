import type { Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";
import { isPhone, pageAction } from "./shell.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 300_000 });

type Alert = "" | " needs attention" | " failed";

async function expectNodesRow(page: Page, alert: Alert, timeout: number): Promise<void> {
  if (!isPhone(page)) {
    await expect(
      page.getByRole("navigation", { name: "Main" }).getByRole("link", { name: /^Nodes/u }),
    ).toHaveText(`Nodes${alert}`, { timeout });
    return;
  }
  await page
    .getByRole("navigation", { name: "Tabs" })
    .getByRole("button", { name: "More" })
    .click();
  const more = page.getByRole("dialog", { name: "More" });
  await expect(more.getByRole("button", { name: /^Nodes/u })).toHaveText(`Nodes${alert}`, {
    timeout,
  });
  await more.getByRole("button", { name: "Cancel" }).click();
  await expect(more).toHaveCount(0);
}

test("an offline seedbox lights the Nodes dot, its Indexer's search says what it waits for, and the dot turns red past the grace", async ({
  adminPage: page,
  world,
}) => {
  await page.goto("/settings/nodes");
  const offline = page.getByRole("region", { name: "Offline Nodes" });
  await offline.getByRole("textbox", { name: "Offline grace" }).fill("2");
  await offline.getByRole("button", { name: "Save" }).click();
  await expect(offline.getByRole("button", { name: "Reset to default" })).toBeVisible();

  await page.goto("/add?contentType=movie");
  await page.getByRole("searchbox", { name: "Title" }).fill("quiet orchard");
  await page.getByRole("main").getByRole("button", { name: "Search", exact: true }).click();
  await page.getByRole("button", { name: "Add The Quiet Orchard" }).click();
  await page
    .getByRole("dialog", { name: "Add The Quiet Orchard" })
    .getByRole("button", { name: "Add Movie" })
    .click();
  await expect(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  await expectNodesRow(page, "", 5000);

  await world.fw.node("stop", "seedbox");
  await expectNodesRow(page, " needs attention", 90_000);

  await pageAction(page, "Search").click();
  await expect(page.getByRole("list", { name: "Searches" })).toContainText(
    /^The Indexer didn't answerWaiting for seedbox · offline since /u,
  );

  await expectNodesRow(page, " failed", 150_000);
});
