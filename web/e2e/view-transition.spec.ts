import { expect, test } from "./fixtures.ts";

declare global {
  interface Window {
    transitionTypes: string[][];
  }
}

test("a forward navigation starts one push transition and Back one pop", async ({
  adminPage: page,
}) => {
  await page.context().addInitScript(() => {
    window.transitionTypes = [];
    const start = document.startViewTransition.bind(document);
    document.startViewTransition = (options) => {
      window.transitionTypes.push(typeof options === "object" ? [...(options.types ?? [])] : []);
      return start(options);
    };
  });
  await page.reload();
  const transitionTypes = () => page.evaluate(() => window.transitionTypes);

  await page.getByRole("link", { name: "This Node" }).click();
  await expect(page.getByRole("heading", { name: "This Node" })).toBeVisible();
  expect(await transitionTypes()).toEqual([["push"]]);

  await page.goBack();
  await expect(page.getByRole("heading", { name: /^Signed in as /u })).toBeVisible();
  expect(await transitionTypes()).toEqual([["push"], ["pop"]]);
});
