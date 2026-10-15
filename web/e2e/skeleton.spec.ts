import { expect, test } from "./fixtures.ts";
import { navigate } from "./shell.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 120_000 });

test("an Admin adds a movie, searches it, and watches it import", async ({
  adminPage: page,
  world,
}) => {
  await page.goto("/add?contentType=movie");
  await page.getByRole("searchbox", { name: "Title" }).fill("quiet orchard");
  await page.getByRole("main").getByRole("button", { name: "Search", exact: true }).click();
  await page.getByRole("button", { name: "Add The Quiet Orchard" }).click();
  await expect(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();

  await page.getByRole("main").getByRole("button", { name: "Search", exact: true }).click();
  await expect(page.getByRole("list", { name: "Searches" })).toHaveText("Grabbed · 2 releases");

  await navigate(page, "Activity");
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
  const changed = await world.fw.sse("admin").list({ where: { event: "grabs.changed" } });
  expect(new Set(changed.map((record) => record.node))).toEqual(new Set(["home"]));
});
