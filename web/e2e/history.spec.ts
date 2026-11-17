import { z } from "zod";

import { expect, expectAccessible, test } from "./fixtures.ts";

test.use({ scenario: "movies" });
test.describe.configure({ timeout: 120_000 });

const Added = z.object({ work: z.object({ id: z.string() }) });

test("History lists a Work added today, opens it in full, shows on its Work page and clears", async ({
  adminPage: page,
  world,
}) => {
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000005",
    preset: "all",
  });
  expect(added.status, "The Salt Road is added").toBe(200);
  const workId = Added.parse(added.body).work.id;

  await page.goto("/activity/history");
  const today = page.getByRole("list", { name: "Today" });
  const entry = today.getByRole("listitem").filter({ hasText: /Added The Salt Road/u });
  await expect(entry).toContainText(/\d{2}:\d{2}/u);
  await entry.getByRole("button", { name: "Details" }).click();
  await expect(page.getByRole("link", { name: /^Open The Salt Road/u })).toHaveAttribute(
    "href",
    `/works/${workId}`,
  );
  await expectAccessible(page);
  await page.keyboard.press("Escape");

  await page.goto(`/works/${workId}`);
  const section = page.getByRole("region", { name: "History" });
  await expect(section).toContainText(/Added The Salt Road/u);
  await section.getByRole("link", { name: "See all in History" }).click();
  await expect(page).toHaveURL(new RegExp(`/activity/history\\?work=${workId}$`, "u"));
  await expect(page.getByRole("button", { name: "Show every Work" })).toBeVisible();

  await page.getByRole("button", { name: "History options" }).click();
  await page.getByRole("menuitem", { name: "Clear History…" }).click();
  await page.getByRole("button", { name: "Clear History", exact: true }).click();
  await expect(page.getByRole("region", { name: "Messages" })).toContainText("Clearing History.");
  await expect(page.getByText("No entries match these filters.")).toBeVisible();
  await page.getByRole("button", { name: "Clear filters" }).click();
  await expect(
    page.getByText(
      "History is empty. Grabs, Imports and Works added or deleted show here as they happen.",
    ),
  ).toBeVisible();
});
