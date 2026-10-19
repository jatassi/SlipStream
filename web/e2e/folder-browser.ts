import type { Page } from "@playwright/test";

import { isPhone } from "./shell.ts";

// A phone shell's Menu is a sheet of buttons.
export const menuItem = (page: Page, menu: string, name: string) =>
  isPhone(page)
    ? page.getByRole("dialog", { name: menu }).getByRole("button", { name, exact: true })
    : page.getByRole("menu", { name: menu }).getByRole("menuitem", { name, exact: true });

export const browser = (page: Page) =>
  page.getByRole("dialog", { name: "Choose a folder on home" });
export const folders = (page: Page) => browser(page).getByRole("listbox", { name: "Folders" });
const filter = (page: Page) =>
  browser(page).getByRole("combobox", { name: "Filter, or go to a path" });

export async function drillInto(page: Page, name: string): Promise<void> {
  await filter(page).fill(name);
  await folders(page).getByRole("option", { name, exact: true }).click();
}

export async function upTo(page: Page, parent: string): Promise<void> {
  await (
    isPhone(page)
      ? browser(page).getByRole("button", { name: parent, exact: true })
      : browser(page)
          .getByRole("navigation", { name: "Folders above" })
          .getByRole("button", { name: parent, exact: true })
  ).click();
}

export async function newFolder(page: Page, name: string): Promise<void> {
  if (isPhone(page)) {
    await browser(page).getByRole("button", { name: "Folder actions" }).click();
    await menuItem(page, "Folder actions", "New folder").click();
  } else {
    await browser(page).getByRole("button", { name: "New folder" }).click();
  }
  await browser(page).getByRole("textbox", { name: "New folder name" }).fill(name);
  await browser(page).getByRole("textbox", { name: "New folder name" }).press("Enter");
}

export const useButton = (page: Page, name: string) =>
  browser(page).getByRole("button", { name: `Use ${name}`, exact: true });
