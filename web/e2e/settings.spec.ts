import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone, navigate } from "./shell.ts";

test("Settings moves between its sections by tabs on desktop and pushed pages on phone", async ({
  adminPage: page,
}) => {
  await navigate(page, "Settings");
  const sections = ["Address", "Email", "Sign-in", "API key", "Targets", "Profiles", "Editions"];
  if (isPhone(page)) {
    const list = page.getByRole("list", { name: "Settings" });
    await expect(list.getByRole("link")).toHaveText(sections);
    await expectAccessible(page);
    await list.getByRole("link", { name: "Address" }).click();
    await expect(page).toHaveURL(/\/settings\/address$/u);
    await expect(page.getByRole("textbox", { name: "Address", exact: true })).toBeVisible();
    await expectAccessible(page);
    await page.getByRole("banner").getByRole("link", { name: "Settings" }).click();
    await expect(page).toHaveURL(/\/settings$/u);
    await expect(list).toBeVisible();
    return;
  }
  await expect(page).toHaveURL(/\/settings\/address$/u);
  const tabs = page.getByRole("navigation", { name: "Settings" });
  await expect(tabs.getByRole("link")).toHaveText(sections);
  await expect(tabs.getByRole("link", { name: "Address" })).toHaveAttribute("aria-current", "page");
  await expectAccessible(page);
  await tabs.getByRole("link", { name: "Email" }).click();
  await expect(page).toHaveURL(/\/settings\/email$/u);
  await expect(tabs.getByRole("link", { name: "Email" })).toHaveAttribute("aria-current", "page");
  await expect(page.getByRole("textbox", { name: "Server", exact: true })).toBeVisible();
  await expectAccessible(page);
});
