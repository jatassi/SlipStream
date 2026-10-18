import { AxeBuilder } from "@axe-core/playwright";
import { type Page, expect } from "@playwright/test";

const wcag = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22a", "wcag22aa"];

export async function expectAccessible(page: Page): Promise<void> {
  // A dialog still fading out measures its buttons against the page. One just
  // opened starts its transition a frame later, so wait two frames first.
  await page.evaluate(async () => {
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          resolve();
        });
      });
    });
    await Promise.allSettled(
      document
        .getAnimations()
        .filter((animation) => animation.effect?.getComputedTiming().endTime !== Infinity)
        .map((animation) => animation.finished),
    );
  });
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
