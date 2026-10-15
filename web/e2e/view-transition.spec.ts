import { expect, test } from "./fixtures.ts";
import { isPhone, navigate } from "./shell.ts";

declare global {
  interface Window {
    transitionTypes: string[][];
    contentAnimations: string[];
  }
}

test("a forward navigation starts one push transition and Back one pop", async ({
  adminPage: page,
}) => {
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
  await page.reload();
  const transitionTypes = () => page.evaluate(() => window.transitionTypes);

  await navigate(page, "Activity");
  await expect(page.getByRole("heading", { name: "Activity" })).toBeVisible();
  expect(await transitionTypes()).toEqual([["push"]]);

  await page.goBack();
  await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(await transitionTypes()).toEqual([["push"], ["pop"]]);
  await expect
    .poll(() => page.evaluate(() => window.contentAnimations))
    .toEqual(isPhone(page) ? ["push-in", "pop-in"] : ["content-fade-in", "content-fade-in"]);
});
