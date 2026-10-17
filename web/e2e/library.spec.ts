import { expect, test } from "./fixtures.ts";
import { navigate } from "./shell.ts";

test("an Admin adds a movie from the catalog and sees its missing cell", async ({
  adminPage: page,
}) => {
  await expect(
    page.getByText("Nothing in the library yet. Add your first Work to start."),
  ).toBeVisible();
  await page.getByRole("main").getByRole("button", { name: "Add a Movie" }).click();
  await page.getByRole("searchbox", { name: "Title" }).fill("quiet orchard");
  await page.getByRole("main").getByRole("button", { name: "Search", exact: true }).click();
  await expect(page.getByRole("list", { name: "Results" })).toHaveText("The Quiet Orchard2021Add");

  await page.getByRole("button", { name: "Add The Quiet Orchard" }).click();
  await page
    .getByRole("dialog", { name: "Add The Quiet Orchard" })
    .getByRole("button", { name: "Add Movie" })
    .click();
  await expect(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Added The Quiet Orchard",
  );
  await expect(page.getByRole("definition")).toHaveText(["2021", "2021-04-16", "1h 44m"]);
  await expect(page.getByRole("columnheader")).toHaveText(["Movie", "1080p"]);
  await expect(page.getByRole("row", { name: /^The Quiet Orchard /u })).toHaveText(
    "The Quiet OrchardMissing",
  );

  await navigate(page, "Library");
  await expect(page.getByRole("list", { name: "Works" }).getByRole("listitem")).toHaveText(
    "The Quiet Orchard2021",
  );
  await page.goto("/add?contentType=movie&query=quiet+orchard");
  await expect(
    page.getByRole("link", { name: "The Quiet Orchard 2021 In the library" }),
  ).toBeVisible();
});
