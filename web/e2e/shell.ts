import type { Page } from "@playwright/test";

export type Destination = "Library" | "Activity" | "Nodes" | "Tasks";

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

// The phone portal has no More sheet: Sign out sits on its Account tab.
export async function signOut(page: Page): Promise<void> {
  const portal = new URL(page.url()).pathname.startsWith("/portal");
  if (isPhone(page) && portal) {
    await page
      .getByRole("navigation", { name: "Tabs" })
      .getByRole("link", { name: "Account" })
      .click();
    await page.getByRole("button", { name: "Sign out of this device" }).click();
  } else if (isPhone(page)) {
    await page
      .getByRole("navigation", { name: "Tabs" })
      .getByRole("button", { name: "More" })
      .click();
    await page
      .getByRole("dialog", { name: "More" })
      .getByRole("button", { name: "Sign out" })
      .click();
  } else {
    await page.getByRole("button", { name: "Account" }).click();
    await page.getByRole("menuitem", { name: "Sign out" }).click();
  }
  const signedOut = page.waitForResponse(
    (response) =>
      response.request().method() === "DELETE" &&
      new URL(response.url()).pathname === "/api/v1/session",
  );
  await page
    .getByRole("dialog", { name: "Sign out of this device?" })
    .getByRole("button", { name: "Sign out" })
    .click();
  await signedOut;
}

export const pageAction = (page: Page, name: string) =>
  page.getByRole(isPhone(page) ? "main" : "banner").getByRole("button", { name, exact: true });
