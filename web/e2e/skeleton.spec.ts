import { expect, test } from "./fixtures.ts";
import { navigate, pageAction } from "./shell.ts";

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
  await page
    .getByRole("dialog", { name: "Add The Quiet Orchard" })
    .getByRole("button", { name: "Add Movie" })
    .click();
  await expect(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();

  await pageAction(page, "Search").click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText("Search started");
  await expect(page.getByRole("list", { name: "Searches" })).toContainText(
    "Grabbed a Release2 Releases foundFixture SiteFound 2 Releases",
  );

  await navigate(page, "Activity");
  const job = page
    .getByRole("table", { name: "Grabs" })
    .getByRole("row", { name: /Quiet Orchard/u });
  await expect(job).toHaveText(
    "The Quiet OrchardThe.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKWWEBDL-1080p1080pImported",
    { timeout: 60_000 },
  );
  await expect(page.getByText("Nothing moving · 1 imported")).toBeVisible();

  await page.getByRole("link", { name: "The Quiet Orchard", exact: true }).click();
  const file = page.getByRole("list", { name: "Files" }).getByRole("listitem");
  await expect(file).toContainText(
    "The Quiet Orchard (2021) - 1080p.mkvThe Quiet Orchard (2021)1920×1080",
  );

  const completed = await world.fw.capture("announces").list({ where: { event: "completed" } });
  expect(completed.map((record) => record.node)).toEqual(["seedbox"]);
  const changed = await world.fw.sse("admin").list({ where: { event: "grabs.changed" } });
  expect(new Set(changed.map((record) => record.node))).toEqual(new Set(["home"]));
});
