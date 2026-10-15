import type { Page } from "@playwright/test";

import { expect, test } from "./fixtures.ts";
import { isPhone, navigate } from "./shell.ts";

declare global {
  interface Window {
    transitionTypes: string[][];
    contentAnimations: string[];
  }
}

async function recordTransitions(page: Page) {
  await page.context().addInitScript(() => {
    window.transitionTypes = [];
    window.contentAnimations = [];
    const start = document.startViewTransition.bind(document);
    document.startViewTransition = (options) => {
      window.transitionTypes.push(typeof options === "object" ? [...(options.types ?? [])] : []);
      const transition = start(options);
      void transition.ready.then(() =>
        window.contentAnimations.push(
          getComputedStyle(document.documentElement, "::view-transition-new(content)")
            .animationName,
        ),
      );
      return transition;
    };
  });
}

const transitionTypes = (page: Page) => page.evaluate(() => window.transitionTypes);
const contentAnimations = (page: Page) => page.evaluate(() => window.contentAnimations);

test("a forward navigation starts one push transition and Back one pop", async ({
  adminPage: page,
}) => {
  await recordTransitions(page);
  await page.reload();

  await navigate(page, "Activity");
  await expect(page.getByRole("heading", { name: "Activity" })).toBeVisible();
  expect(await transitionTypes(page)).toEqual([["push"]]);

  await page.goBack();
  await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(await transitionTypes(page)).toEqual([["push"], ["pop"]]);
  await expect
    .poll(() => contentAnimations(page))
    .toEqual(isPhone(page) ? ["push-in", "pop-in"] : ["content-fade-in", "content-fade-in"]);
});

test("a link up to the page above pops back to it", async ({ adminPage: page }) => {
  await recordTransitions(page);
  await page.goto("/add?contentType=movie");
  await expect(page.getByRole("heading", { name: "Add a Movie" })).toBeVisible();

  await page.getByRole("banner").getByRole("link", { name: "Movies" }).click();
  await expect(page.getByRole("heading", { name: "Movies", level: 1 })).toBeVisible();
  expect(await transitionTypes(page)).toEqual([["pop"]]);
  await expect
    .poll(() => contentAnimations(page))
    .toEqual([isPhone(page) ? "pop-in" : "content-fade-in"]);
});
