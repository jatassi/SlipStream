import type { Page } from "@playwright/test";

import { expect } from "./fixtures.ts";

// A new Controller warns in Health until its Admin confirms saving the Backup
// passphrase, which only a signed-in session can do.
export async function confirmPassphraseSaved(page: Page): Promise<void> {
  const status = await page.evaluate(
    async () => (await fetch("/api/v1/backups/passphrase/saved", { method: "POST" })).status,
  );
  expect(status).toBe(204);
}
