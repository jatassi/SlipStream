import { type BrowserContext, type Page, expect } from "@playwright/test";

const sheets = "[role=dialog][data-swipe-direction]";

// A sheet slides up from below the screen as it opens. Under load the page can
// stall before the slide starts, and Playwright's stable check reads the frames
// either side of the stall as a sheet at rest, so it presses a row that is
// still sliding and the release lands beside it. A person waits for the sheet.
async function arrive(page: Page): Promise<void> {
  const open = page.locator(sheets);
  await expect(open.and(page.locator("[data-starting-style]"))).toHaveCount(0);
  await open.evaluateAll(async (popups) => {
    await Promise.allSettled(
      popups.flatMap((popup) => popup.getAnimations()).map((animation) => animation.finished),
    );
  });
}

export function waitForSheets(context: BrowserContext): void {
  const newPage = context.newPage.bind(context);
  context.newPage = async () => {
    const page = await newPage();
    await page.addLocatorHandler(page.locator(sheets).last(), () => arrive(page), {
      noWaitAfter: true,
    });
    return page;
  };
}
