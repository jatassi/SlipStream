import { AxeBuilder } from "@axe-core/playwright";
import { type Page, expect } from "@playwright/test";

const wcag = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22a", "wcag22aa"];

export async function expectAccessible(page: Page): Promise<void> {
  // A dialog still fading out measures its buttons against the page. Base UI
  // can hold a popup in data-starting-style for several frames before its
  // transition starts, so wait for two frames in a row with nothing entering,
  // leaving or animating.
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
  // A row scrolled under the sticky top bar fails target-size, wherever the
  // last click left the page.
  await page.evaluate(() => {
    window.scrollTo(0, 0);
  });
  // Base UI gives its focus guards role="button" in WebKit, so VoiceOver can
  // land on them; they are focus traps, not commands, and have no name.
  const { violations } = await new AxeBuilder({ page })
    .withTags(wcag)
    .exclude("[data-base-ui-focus-guard]")
    .analyze();
  const found = violations.map(
    (violation) =>
      `${violation.id}: ${violation.nodes.map((node) => node.target.join(" ")).join(", ")}`,
  );
  expect(found, `axe on ${page.url()}`).toEqual([]);
}
