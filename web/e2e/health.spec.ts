import type { Locator, Page } from "@playwright/test";
import { z } from "zod";

import { menuItem } from "./folder-browser.ts";
import { type World, expect, test } from "./fixtures.ts";
import { isPhone, navigate } from "./shell.ts";

test.describe.configure({ timeout: 300_000 });

type Alert = "" | " needs attention" | " failed";

const Volumes = z.object({ volumes: z.array(z.object({ name: z.string() })) });

async function homeVolume(world: World): Promise<string> {
  const reply = await world.api("GET", "/api/v1/system/health/volumes");
  const [volume] = Volumes.parse(reply.body).volumes;
  if (volume === undefined) {
    throw new Error("no Node reports the volume of the Movies root");
  }
  return volume.name;
}

async function expectHealthRow(page: Page, alert: Alert, timeout = 5000): Promise<void> {
  if (!isPhone(page)) {
    await expect(
      page.getByRole("navigation", { name: "Main" }).getByRole("link", { name: /^Health/u }),
    ).toHaveText(`Health${alert}`, { timeout });
    return;
  }
  await page
    .getByRole("navigation", { name: "Tabs" })
    .getByRole("button", { name: "More" })
    .click();
  const more = page.getByRole("dialog", { name: "More" });
  await expect(more.getByRole("button", { name: /^Health/u })).toHaveText(`Health${alert}`, {
    timeout,
  });
  await more.getByRole("button", { name: "Cancel" }).click();
  await expect(more).toHaveCount(0);
}

const category = (page: Page, label: string) => page.getByRole("region", { name: label });

const checked = (scope: Page | Locator) =>
  scope.getByRole("status").filter({ hasText: /^Checked\./u });

const itemRow = (scope: Locator, text: string) =>
  scope.getByRole("button").filter({ hasText: text });

async function openItem(
  page: Page,
  scope: Locator,
  text: string,
  subject: string,
): Promise<Locator> {
  await itemRow(scope, text).click();
  return isPhone(page) ? page.getByRole("dialog", { name: subject }) : scope;
}

async function closeItem(page: Page, opened: Locator): Promise<void> {
  if (isPhone(page)) {
    await page.keyboard.press("Escape");
    await expect(opened).toHaveCount(0);
  }
}

async function setWarningLevels(page: Page, volume: string): Promise<void> {
  await page.getByRole("button", { name: `Actions for ${volume}`, exact: true }).click();
  await menuItem(page, volume, "Set warning levels").click();
}

const storageSummary = (warn: string, error: string) =>
  `A volume shows a warning below ${warn} free and an error below ${error}. Storage Nodes check each volume every 5 min, and again after each import.`;

async function calmPage(page: Page): Promise<void> {
  await expectHealthRow(page, "");
  await navigate(page, "Health");
  await expect(page.getByText("Everything is working.")).toBeVisible();
  await expect(page.getByText("Checked storage, Nodes and scheduled tasks")).toBeVisible();
  for (const label of ["Storage", "Nodes", "Scheduled tasks"]) {
    // oxlint-disable-next-line no-await-in-loop -- one quiet line per category, in page order
    await expect(category(page, label)).toContainText(`${label}. All good.`);
  }
  await expect(
    category(page, "Scheduled tasks").getByRole("button", { name: /^Check now/u }),
  ).toHaveCount(0);

  await page.getByRole("button", { name: "Check all" }).click();
  await expect(checked(page)).toHaveText("Checked.");
  const nodes = category(page, "Nodes");
  await nodes.getByRole("button", { name: "Check now: Nodes" }).click();
  await expect(checked(nodes)).toHaveText("Checked.");
  const storage = category(page, "Storage");
  await storage.getByRole("button", { name: "Check now: Storage" }).click();
  await expect(checked(storage)).toHaveText("Checked.");
  await expect(page.getByText("Everything is working.")).toBeVisible();
}

async function levelsOnHealth(page: Page): Promise<void> {
  await navigate(page, "Health");
  const storage = category(page, "Storage");
  await expect(storage).toContainText(storageSummary("0 GB", "0 GB"));
  await storage.getByRole("button", { name: "Details" }).click();

  const warn = storage.getByRole("textbox", { name: "Warn when free space is below" });
  const error = storage.getByRole("textbox", { name: "Show an error when free space is below" });
  await storage.getByRole("radio", { name: "%" }).first().click();
  await warn.fill("2");
  await storage.getByRole("button", { name: "Save" }).click();
  await expect(storage).toContainText(storageSummary("2%", "0 GB"));

  await storage.getByRole("radio", { name: "%" }).nth(1).click();
  await error.fill("5");
  await storage.getByRole("button", { name: "Save" }).click();
  await expect(storage.getByRole("alert")).toHaveText("No more than the warning level, 2%.");
  await expect(storage).toContainText(storageSummary("2%", "0 GB"));

  await warn.fill("100");
  await storage.getByRole("button", { name: "Save" }).first().click();
  await expect(storage).toContainText(storageSummary("100%", "0 GB"));
  await expect(itemRow(storage, "Free up space on ")).toContainText("Warning");
  await expect(page.getByRole("link", { name: /^Free up space on /u })).toBeVisible();
  await expectHealthRow(page, " needs attention");
}

async function volumeLevels(page: Page, world: World): Promise<void> {
  const name = await homeVolume(world);
  await page.goto("/settings/library-roots");
  await setWarningLevels(page, name);
  const dialog = page.getByRole("dialog", { name: `Warning levels for ${name}` });
  await expect(dialog).toContainText(
    `${name} on home follows the global levels until you change one here.`,
  );
  await dialog.getByRole("radio", { name: "%" }).first().click();
  await dialog.getByRole("textbox", { name: "Warn when free space is below" }).fill("100");
  await dialog.getByRole("button", { name: "Save" }).click();
  await expect(dialog).toContainText(`${name} on home has its own levels.`);
  await dialog.getByRole("button", { name: "Done" }).click();
  await expect(dialog).toHaveCount(0);

  const volumes = page.getByRole("list", { name: "Volumes on home" });
  await expect(volumes).toContainText("Own levels");
  await expect(volumes).toContainText("Warning");
  const warning = page.getByRole("button", {
    name: isPhone(page) ? `${name} on home` : "Open Health",
  });
  await expect(warning).toBeVisible();
  if (!isPhone(page)) {
    await expect(page.getByRole("link", { name: "Library roots 1 warning" })).toBeVisible();
  }

  await setWarningLevels(page, name);
  await dialog.getByRole("button", { name: "Follow the global levels" }).click();
  await expect(dialog).toContainText(
    `${name} on home follows the global levels until you change one here.`,
  );
  await dialog.getByRole("button", { name: "Done" }).click();
  await expect(volumes).not.toContainText("Own levels");
  await expect(volumes).not.toContainText("Warning");
  await expect(warning).toHaveCount(0);
}

async function lowSpace(page: Page, world: World): Promise<void> {
  const subject = `${await homeVolume(world)} on home`;
  await expectHealthRow(page, " failed");
  if (isPhone(page)) {
    await page.goto("/settings");
    await page.getByRole("link", { name: "Library roots 1 error" }).click();
    await page.getByRole("button").filter({ hasText: subject }).click();
    const sheet = page.getByRole("dialog", { name: subject });
    await expect(sheet).toContainText("Error");
    await sheet.getByRole("button", { name: "Open Health" }).click();
  } else {
    await page.goto("/settings/library-roots");
    await expect(page.getByRole("link", { name: "Library roots 1 error" })).toBeVisible();
    await page.getByRole("button", { name: "Open Health" }).click();
  }
  await expect(page).toHaveURL(/\/health#health-storage-/u);

  await expect(page.getByRole("link", { name: /^Free up space on /u })).toBeVisible();
  const storage = category(page, "Storage");
  const opened = await openItem(page, storage, subject, subject);
  await expect(opened).toContainText("Error level");
  await expect(opened).toContainText("1024 TB (global)");
  await opened.getByRole("button", { name: `Check now: ${subject}` }).click();
  await expect(checked(opened)).toHaveText("Checked.");
  await closeItem(page, opened);
  await expect(itemRow(storage, subject)).toContainText("Error");
}

async function nodeOffline(page: Page, world: World): Promise<void> {
  const grace = await world.api("PATCH", "/api/v1/nodes/settings", { offlineGrace: 60 });
  expect(grace.status).toBe(200);
  await navigate(page, "Health");
  await expect(page.getByText("Everything is working.")).toBeVisible();

  await world.fw.node("stop", "seedbox");
  const nodes = category(page, "Nodes");
  const offline = "seedbox has been offline past the offline grace.";
  await expect(itemRow(nodes, offline)).toContainText("Error", { timeout: 150_000 });
  await expect(page.getByRole("link", { name: new RegExp(`^${offline}`, "u") })).toBeVisible();
  await expect(itemRow(nodes, "Searching on home")).toContainText("Warning");
  await expectHealthRow(page, " failed");

  const opened = await openItem(page, nodes, offline, "seedbox");
  await expect(opened).toContainText("Offline since");
  await opened.getByRole("button", { name: "Open seedbox" }).click();
  await expect(page.getByRole("heading", { name: "seedbox", level: 1 })).toBeVisible();
  await expect(page).toHaveURL(/\/nodes\/[^/]+$/u);
  await navigate(page, "Health");

  await nodes.getByRole("button", { name: "Check now: Nodes" }).click();
  await expect(checked(nodes)).toHaveText("Checked. seedbox is offline.");

  await world.fw.node("start", "seedbox");
  await expect(itemRow(nodes, offline)).toHaveCount(0, { timeout: 60_000 });
  await expect(page.getByText("Everything is working.")).toBeVisible({ timeout: 150_000 });
  await expectHealthRow(page, "");
}

test.describe("with nothing wrong", () => {
  test.use({ scenario: "single-machine" });

  test("the Health page says everything is working, names what it checked, and Check now and Check all answer Checked", async ({
    adminPage,
  }) => {
    await calmPage(adminPage);
  });

  test("the Storage levels save on the Health page, a same-unit error level above the warning is refused, and 100% raises a warning", async ({
    adminPage,
  }) => {
    await levelsOnHealth(adminPage);
  });

  test("Set warning levels gives a Library roots volume its own levels, its row and the section turn amber, and Follow the global levels clears them", async ({
    adminPage,
    world,
  }) => {
    await volumeLevels(adminPage, world);
  });
});

test.describe("with a volume below its levels", () => {
  test.use({ scenario: "low-space" });

  test("Library roots carries the red dot and opens with the volume's error, and Open Health leads to it", async ({
    adminPage,
    world,
  }) => {
    await lowSpace(adminPage, world);
  });
});

test.describe("with a Worker away", () => {
  test.use({ scenario: "home" });

  test("seedbox offline past its grace is a Nodes error with the red Health dot, Check now says it is offline, and it clears once seedbox is back", async ({
    adminPage,
    world,
  }) => {
    await nodeOffline(adminPage, world);
  });
});
