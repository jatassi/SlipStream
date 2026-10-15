import { page, server } from "vitest/browser";

let resized = false;

// Resizing skips a running view transition, and the router leaves the skipped
// transition's ready promise rejected and unhandled, so every resize waits for
// the last navigation's transition, even one an earlier test started.
async function setViewport(width: number, height: number) {
  await document.activeViewTransition?.finished;
  await page.viewport(width, height);
}

export async function resize(width: number, height: number) {
  resized = true;
  await setViewport(width, height);
}

export const asPhone = () => resize(390, 844);

/** Back to the configured desktop preset once the test's apps have unmounted. */
export async function restoreViewport() {
  if (!resized) {
    return;
  }
  resized = false;
  const { width, height } = server.config.browser.viewport;
  await setViewport(width, height);
}
