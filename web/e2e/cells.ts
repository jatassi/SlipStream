import type { Page } from "@playwright/test";

import { expect } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

const standard = (page: Page) => page.getByRole("table", { name: "Standard cells" });

// On phone the table stacks each row as a card whose labels join its name, so a row is found by its text.
export const orchardCells = (page: Page) =>
  standard(page).getByRole("row").filter({ hasText: "The Quiet Orchard" });

// A stacked table on phone has no column headers, only each cell's label.
export async function expectColumns(page: Page, headers: readonly string[]): Promise<void> {
  if (!isPhone(page)) {
    await expect(standard(page).getByRole("columnheader")).toHaveText([...headers]);
  }
}
