import type { Page } from "@playwright/test";

export type Destination = "Library" | "Activity" | "This Node";

export const isPhone = (page: Page): boolean => (page.viewportSize()?.width ?? 0) < 768;

export const shellNav = (page: Page) =>
  page.getByRole("navigation", { name: isPhone(page) ? "Tabs" : "Main" });

// A nav row's name carries its count, as in "Activity 2 live".
const named = (label: string) => new RegExp(`^${label}\\b`, "u");

export async function navigate(page: Page, to: Destination): Promise<void> {
  await shellNav(page).waitFor();
  if (!isPhone(page)) {
    await page
      .getByRole("navigation", { name: "Main" })
      .getByRole("link", { name: named(to) })
      .click();
    return;
  }
  const tab = page.getByRole("navigation", { name: "Tabs" }).getByRole("link", { name: named(to) });
  if ((await tab.count()) > 0) {
    await tab.click();
    return;
  }
  await page
    .getByRole("navigation", { name: "Tabs" })
    .getByRole("button", { name: "More" })
    .click();
  await page
    .getByRole("dialog", { name: "More" })
    .getByRole("button", { name: named(to) })
    .click();
}

export async function signOut(page: Page): Promise<void> {
  if (isPhone(page)) {
    await page
      .getByRole("navigation", { name: "Tabs" })
      .getByRole("button", { name: "More" })
      .click();
    await page
      .getByRole("dialog", { name: "More" })
      .getByRole("button", { name: "Sign out" })
      .click();
    return;
  }
  await page.getByRole("button", { name: "Account" }).click();
  await page.getByRole("menuitem", { name: "Sign out" }).click();
}
