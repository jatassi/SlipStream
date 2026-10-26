import type { Page } from "@playwright/test";
import { z } from "zod";

import { workAction } from "./cells.ts";
import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

test.use({ scenario: "movies" });
test.describe.configure({ timeout: 120_000 });

const Added = z.object({ work: z.object({ id: z.string() }) });
const notifications = (page: Page) => page.getByRole("region", { name: "Notifications" });

test("Refresh metadata reads the source again, and a Work the source drops says so", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000005",
    monitor: true,
  });
  expect(added.status, "The Salt Road is added").toBe(200);
  const workId = Added.parse(added.body).work.id;
  await page.goto(`/works/${workId}`);
  await expect(page.getByRole("heading", { name: "The Salt Road", level: 1 })).toBeVisible();
  await expect(page.getByRole("main")).toContainText("Two sisters walk the salt road.");

  await world.fw.cue("tmdb-change", { id: 9_000_005 });
  await workAction(page, "Refresh metadata", "The Salt Road");
  await expect(notifications(page)).toContainText("Refreshed the metadata of The Salt Road.");
  await expect(page.getByRole("main")).toContainText("Two sisters walk the salt road. Revision 1.");

  await world.fw.cue("tmdb-delete", { id: 9_000_005 });
  await workAction(page, "Refresh metadata", "The Salt Road");
  await expect(notifications(page)).toContainText(
    "TMDB no longer lists The Salt Road, so it wasn't refreshed.",
  );
  await expect(page.getByText("Refresh metadata is unavailable")).toBeVisible();
  await expect(
    page.getByText("TMDB no longer lists this movie. SlipStream keeps it as it is."),
  ).toBeVisible();
  await expect(page.getByRole("main")).toContainText("Two sisters walk the salt road. Revision 1.");
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath("gone-upstream.png"), fullPage: true });

  await page.getByRole("button", { name: "More actions for The Salt Road" }).click();
  await expect(
    page.getByRole(isPhone(page) ? "button" : "menuitem", { name: "Refresh metadata" }),
  ).toBeDisabled();
});
