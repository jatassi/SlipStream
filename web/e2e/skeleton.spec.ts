import { expect, test } from "./fixtures.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 120_000 });

test("an Admin adds a movie, searches it, and watches it import", async ({
  adminPage: page,
  world,
}) => {
  await page.getByRole("link", { name: "Add movie" }).click();
  await page.getByRole("searchbox", { name: "Title" }).fill("quiet orchard");
  await page.getByRole("button", { name: "Search" }).click();
  await page.getByRole("button", { name: "Add The Quiet Orchard" }).click();
  await expect(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();

  await page.getByRole("button", { name: "Search" }).click();
  await expect(page.getByRole("list", { name: "Searches" })).toHaveText("Grabbed · 2 releases");

  await page.getByRole("link", { name: "Activity" }).click();
  await expect(page.getByRole("list", { name: "Grabs" }).getByRole("listitem")).toHaveText(
    "The Quiet OrchardThe.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKWImported",
    { timeout: 60_000 },
  );

  await page.getByRole("link", { name: "The Quiet Orchard" }).click();
  await expect(page.getByRole("list", { name: "Files" })).toHaveText(
    "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv 1920×1080",
  );

  const completed = await world.fw.capture("announces").list({ where: { event: "completed" } });
  expect(completed.map((record) => record.node)).toEqual(["seedbox"]);
});
