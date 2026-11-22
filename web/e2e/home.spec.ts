import { expect, expectAccessible, test } from "./fixtures.ts";
import { grabOnSeedbox } from "./grabs.ts";
import { isPhone } from "./shell.ts";
import { screenshotWholePage } from "./whole-page.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 180_000 });

test("a Grab moves through Jobs and arrives in Recent with no reload", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await expect(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  await expect(page.getByText(/^No jobs are moving\./u)).toBeVisible();
  await world.fw.fault.set({ id: "swarm-stalled", target: "seeders", fault: "stall" });
  await grabOnSeedbox(world);

  const job = page.getByRole("table", { name: "Jobs" }).getByRole("row", { name: /Quiet/u });
  await expect(job.getByText("Downloading", { exact: true })).toBeVisible({ timeout: 90_000 });
  await world.fw.fault.clear("swarm-stalled");
  await expect(job.getByText("Importing", { exact: true })).toBeVisible({ timeout: 90_000 });
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("home-importing.png"));

  await expect(job).toHaveCount(0, { timeout: 90_000 });
  const recent = page.getByRole("list", { name: "Recent" });
  await expect(recent.getByRole("link", { name: /^The Quiet Orchard/u })).toContainText(
    "Imported in",
  );
  await expect(page.getByText(/^Today: 1 imported$/u)).toBeVisible({ visible: !isPhone(page) });
});
