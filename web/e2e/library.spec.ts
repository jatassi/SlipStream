import { expect, test } from "./fixtures.ts";

test("an Admin adds a movie from the catalog and sees its missing cell", async ({
  adminPage: page,
}) => {
  await page.getByRole("link", { name: "Add movie" }).click();
  await page.getByRole("searchbox", { name: "Title" }).fill("quiet orchard");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(page.getByRole("list", { name: "Results" })).toHaveText("The Quiet Orchard 2021Add");

  await page.getByRole("button", { name: "Add The Quiet Orchard" }).click();
  await expect(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  await expect(page.getByRole("definition")).toHaveText(["2021", "2021-04-16", "1h 44m"]);
  await expect(page.getByRole("columnheader")).toHaveText(["Movie", "1080p"]);
  await expect(page.getByRole("row", { name: /^The Quiet Orchard /u })).toHaveText(
    "The Quiet OrchardMissing",
  );

  await page.getByRole("link", { name: "Library" }).click();
  await expect(page.getByRole("main").getByRole("listitem")).toHaveText("The Quiet Orchard 2021");
  await page.goto("/add?query=quiet+orchard");
  await expect(page.getByRole("link", { name: "Open The Quiet Orchard" })).toBeVisible();
});
