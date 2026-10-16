import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, test } from "./fixtures.ts";
import { isPhone, navigate, pageAction } from "./shell.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 120_000 });

// A table row on desktop, a list row on phone.
const housekeepingRow = (page: Page) =>
  isPhone(page)
    ? page
        .getByRole("list", { name: "Controller" })
        .getByRole("listitem")
        .filter({ hasText: "Housekeeping" })
    : page.getByRole("table", { name: "Controller" }).getByRole("row", { name: /Housekeeping/u });

const runs = (page: Page) => page.getByRole("list", { name: "Runs" }).getByRole("listitem");

const seedboxUnit = (page: Page) =>
  page
    .getByRole("table", { name: "Units" })
    .getByRole("row", { name: /Local database on seedbox/u });

const progress = /home, seedbox · \d of 6 steps/u;

// Housekeeping finishes in well under a second, so a latency on the Node
// link holds its seedbox unit open while the page shows the run.
async function slowSeedbox(world: World): Promise<() => Promise<unknown>> {
  await world.fw.fault.set({ id: "slow-link", target: "link/home", fault: "latency=1500ms" });
  return () => world.fw.fault.clear("slow-link");
}

test("Tasks shows Housekeeping in the Controller group with its cadence, next run and Nodes", async ({
  adminPage: page,
}) => {
  await navigate(page, "Tasks");
  await expect(page.getByRole("heading", { level: 2, name: "Controller" })).toBeVisible();
  const row = housekeepingRow(page);
  // Between 04:00 and 06:00 the window is open, so the next one is tomorrow's.
  await expect(row).toContainText(/(Tonight|Tomorrow) at 04:00/u);
  if (isPhone(page)) {
    return;
  }
  await expect(row).toContainText("Daily in the Maintenance window");
  await expect(page.getByRole("columnheader", { name: "Nodes" })).toBeVisible();
  await expect(row).toContainText("home, seedbox");
});

async function expectRunningShown(page: Page): Promise<void> {
  if (isPhone(page)) {
    await navigate(page, "Activity");
    const running = page.getByRole("region", { name: "Running tasks" });
    await expect(running).toContainText("Housekeeping");
    await expect(running).toContainText(progress);
    return;
  }
  const capsule = page.getByRole("banner").getByRole("button", { name: /running/u });
  await expect(capsule).toHaveAccessibleName("1 task running: Housekeeping");
  await capsule.click();
  await expect(page.getByRole("dialog", { name: "Running tasks" })).toContainText(progress);
  await page.keyboard.press("Escape");
}

test("Run now shows the run at once, in the capsule or atop Activity, and it ends Succeeded", async ({
  adminPage: page,
  world,
}) => {
  await navigate(page, "Tasks");
  const healLink = await slowSeedbox(world);
  await housekeepingRow(page).getByRole("button", { name: "Run now: Housekeeping" }).click();
  await expectRunningShown(page);
  await healLink();
  await expect(page.getByRole("region", { name: "Running tasks" })).toHaveCount(0, {
    timeout: 30_000,
  });
  await expect(page.getByRole("button", { name: /task running/u })).toHaveCount(0);

  await navigate(page, "Tasks");
  await expect(housekeepingRow(page)).toContainText("Succeeded");
  await page.getByRole("link", { name: "Housekeeping", exact: true }).click();
  await expect(runs(page).first()).toContainText(/^Succeeded6 stepsRun now/u);
});

test("with seedbox stopped Run now is Partial, and its return runs seedbox's unit", async ({
  adminPage: page,
  world,
}) => {
  await world.fw.node("stop", "seedbox");
  await page.goto("/tasks/housekeeping");
  await pageAction(page, "Run now: Housekeeping").click();
  await expect(runs(page).first()).toContainText(/^Partial5 of 6 steps, seedbox offlineRun now/u, {
    timeout: 30_000,
  });
  await expect(seedboxUnit(page)).toContainText(
    "SkippedNeverseedbox is offline. Runs when seedbox returns.",
  );

  await world.fw.node("start", "seedbox");
  await expect(runs(page).first()).toContainText(/^Succeeded1 stepNode returned/u, {
    timeout: 60_000,
  });
  await expect(seedboxUnit(page)).toContainText(/Succeeded\d\d:\d\d/u);
  await runs(page).first().getByRole("button").click();
  await expect(runs(page).first().getByRole("list", { name: "Units in this run" })).toHaveText(
    "Local database on seedboxseedboxSucceeded",
  );
});

async function setEveryTwoDays(page: Page, detail: Page): Promise<void> {
  if (isPhone(page)) {
    const when = detail.getByRole("region", { name: "When it runs" });
    await when.getByRole("textbox", { name: "Runs every" }).fill("2");
    await when.getByRole("button", { name: "Save" }).click();
    return;
  }
  await housekeepingRow(page)
    .getByRole("button", { name: "Runs every: Daily in the Maintenance window" })
    .click();
  const popover = page.getByRole("dialog", { name: "Runs every" });
  await popover.getByRole("textbox", { name: "Runs every" }).fill("2");
  await popover.getByRole("button", { name: "Save" }).click();
  await expect(housekeepingRow(page)).toContainText("Every 2 days in the Maintenance window");
}

test("a cadence set on one surface reads on the other, and Reset brings back daily", async ({
  adminPage: page,
  context,
}) => {
  await page.goto("/tasks");
  const detail = await context.newPage();
  await detail.goto("/tasks/housekeeping");
  const when = detail.getByRole("region", { name: "When it runs" });
  await expect(when).toContainText("Daily in the Maintenance window");

  await setEveryTwoDays(page, detail);
  await expect(when).toContainText("Every 2 days in the Maintenance window");
  await expect(when.getByRole("textbox", { name: "Runs every" })).toHaveValue("2");
  await when.getByRole("button", { name: "Reset to default" }).click();
  await expect(when).toContainText("Daily in the Maintenance window");
  await expect(when.getByRole("textbox", { name: "Runs every" })).toHaveValue("1");
  await expect(when.getByText("day", { exact: true })).toBeVisible();
  await expect(housekeepingRow(page)).toContainText(/(Tonight|Tomorrow) at 04:00/u);
});

const Schedule = z.object({ window: z.object({ zone: z.string() }) });

test("a Maintenance window of 01:00 to 03:00 shows its zone and moves Housekeeping to 01:00", async ({
  adminPage: page,
  world,
}) => {
  const { zone } = Schedule.parse((await world.api("GET", "/api/v1/tasks/settings")).body).window;
  await navigate(page, "Tasks");
  const window = page.getByRole("group", { name: "Maintenance window" });
  await expect(window).toContainText(zone);
  await window.getByLabel("From").fill("01:00");
  await window.getByLabel("to").fill("03:00");
  await window.getByRole("button", { name: "Save" }).click();
  await expect(window.getByRole("button", { name: "Save" })).toHaveCount(0);
  await expect(window.getByLabel("From")).toHaveValue("01:00");
  await expect(window).toContainText(zone);
  await expect(housekeepingRow(page)).toContainText(/at 01:00/u);
});
