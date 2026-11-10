import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, navigate } from "./shell.ts";

const groups = [
  ["Library", ["Content Types", "Library roots", "Library scan", "Naming", "Metadata"]],
  ["Files to keep", ["Targets", "Profiles", "Custom Formats", "Editions"]],
  ["Search and download", ["Indexers", "Trackers", "Download Clients"]],
  ["Access", ["Address", "Sign-in", "API key"]],
  ["System", ["Nodes", "Email"]],
] as const;

test("Settings lists its sections by group, in a nav on desktop and pushed pages on phone", async ({
  adminPage: page,
}) => {
  await navigate(page, "Settings");
  if (isPhone(page)) {
    await expect(page.getByRole("heading", { level: 2 })).toHaveText(
      groups.map(([group]) => group),
    );
    await Promise.all(
      groups.map(([group, sections]) =>
        expect(page.getByRole("list", { name: group }).getByRole("link")).toHaveText(sections),
      ),
    );
    await expectAccessible(page);
    await page.getByRole("list", { name: "Access" }).getByRole("link", { name: "Address" }).click();
    await expect(page).toHaveURL(/\/settings\/address$/u);
    await expect(page.getByRole("textbox", { name: "Address", exact: true })).toBeVisible();
    await expectAccessible(page);
    await page.getByRole("banner").getByRole("link", { name: "Settings" }).click();
    await expect(page).toHaveURL(/\/settings$/u);
    await expect(page.getByRole("list", { name: "Access" })).toBeVisible();
    return;
  }
  await expect(page).toHaveURL(/\/settings\/content-types$/u);
  const nav = page.getByRole("navigation", { name: "Settings" });
  await expect(nav.getByRole("list")).toHaveCount(groups.length);
  await Promise.all(
    groups.map(async ([group, sections], index) => {
      const list = nav.getByRole("list").nth(index);
      await expect(list).toHaveAccessibleName(group);
      await expect(list.getByRole("link")).toHaveText(sections);
    }),
  );
  await expect(nav.getByRole("link", { name: "Content Types" })).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expectAccessible(page);
  await nav.getByRole("link", { name: "Email" }).click();
  await expect(page).toHaveURL(/\/settings\/email$/u);
  await expect(nav.getByRole("link", { name: "Email" })).toHaveAttribute("aria-current", "page");
  await expect(page.getByRole("textbox", { name: "Server", exact: true })).toBeVisible();
  await expectAccessible(page);
});
