import type { Locator, Page } from "@playwright/test";

// Base UI can hold a popup in data-starting-style for several frames before
// its transition starts, so wait for two frames in a row with nothing
// entering, leaving or animating.
export async function untilStill(page: Page): Promise<void> {
  await page.evaluate(
    () =>
      new Promise<void>((resolve) => {
        let still = 0;
        const frame = () => {
          const moving =
            document.querySelector("[data-starting-style], [data-ending-style]") !== null ||
            document
              .getAnimations()
              .some(
                (animation) =>
                  animation.playState === "running" &&
                  animation.effect?.getComputedTiming().endTime !== Infinity,
              );
          still = moving ? 0 : still + 1;
          if (still === 2) {
            resolve();
          } else {
            requestAnimationFrame(frame);
          }
        };
        requestAnimationFrame(frame);
      }),
  );
}

// WebKit can hold an entering sheet at its start for frames, which passes the
// click's stability check; the tap then lands while the sheet slides and its
// press and release fall on different rows.
export async function clickWhenStill(row: Locator): Promise<void> {
  await row.waitFor();
  await untilStill(row.page());
  await row.click();
}
