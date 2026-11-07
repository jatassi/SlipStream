import type { Page } from "@playwright/test";

// Chromium's fullPage capture lays the page out at 1x1 for a moment. If the app
// renders then, useShellKind picks the phone shell, which remounts the page and
// drops any unsaved draft. Growing only the height keeps every width breakpoint.
export async function screenshotWholePage(page: Page, path: string): Promise<void> {
  const viewport = page.viewportSize();
  if (viewport === null) {
    throw new Error("the page has no fixed viewport to grow");
  }
  const height = await page.evaluate(() => document.documentElement.scrollHeight);
  await page.setViewportSize({ width: viewport.width, height: Math.max(height, viewport.height) });
  try {
    await page.screenshot({ path });
  } finally {
    await page.setViewportSize(viewport);
  }
}
