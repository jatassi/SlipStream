import type { Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";

test.use({ startupSequence: true });

const sequenceOf = (page: Page) => page.getByRole("img", { name: "SlipStream" });

async function drawing(page: Page): Promise<string> {
  await page.waitForFunction(() => document.fonts.check('600 16px "Geist Variable"'));
  await page.evaluate(
    () =>
      new Promise((resolve) => {
        requestAnimationFrame(() => {
          requestAnimationFrame(resolve);
        });
      }),
  );
  return sequenceOf(page).innerHTML();
}

test("a new tab plays the startup sequence over a working page until a key skips it", async ({
  page,
  world,
}) => {
  const sequence = sequenceOf(page);
  await page.goto("/sign-in");
  await expect(sequence).toBeVisible();
  const first = await drawing(page);

  await page.getByLabel("Email").fill(world.admin().email);
  expect(await drawing(page)).not.toBe(first);
  await expect(sequence).toBeVisible();

  await page.keyboard.press("Shift");
  await expect(sequence).toBeHidden();
  await expect(page.getByLabel("Email")).toHaveValue(world.admin().email);
  const resources = await page.evaluate(() =>
    performance.getEntriesByType("resource").map((entry) => new URL(entry.name).pathname),
  );
  expect(resources.filter((path) => /^\/assets\/grain-[\w-]+\.png$/u.test(path))).toHaveLength(1);

  await page.reload();
  await expect(page.getByRole("heading", { name: "Sign in to SlipStream" })).toBeVisible();
  await expect(sequence).toHaveCount(0);
});

test("under reduced motion the finished lockup holds still, then leaves", async ({ page }) => {
  const sequence = sequenceOf(page);
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/sign-in");
  await expect(sequence).toBeVisible();

  const first = await drawing(page);
  expect(await drawing(page)).toBe(first);
  await expect(sequence).toBeHidden({ timeout: 2000 });
});
