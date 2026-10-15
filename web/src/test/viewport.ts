import { page } from "vitest/browser";

import presets from "../../harness/presets.json" with { type: "json" };

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

export const asPhone = () => resize(presets.phone.width, presets.phone.height);

/** Back to the desktop preset once the test's apps have unmounted. */
export async function restoreViewport() {
  if (!resized) {
    return;
  }
  resized = false;
  await setViewport(presets.desktop.width, presets.desktop.height);
}
