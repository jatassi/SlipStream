import { expect, test } from "./fixtures.ts";

test("fonts and icons come from the Node", async ({ adminPage: page, world }) => {
  const fonts = await page.evaluate(async () => {
    await document.fonts.ready;
    return {
      check: document.fonts.check('16px "Geist Variable"'),
      body: getComputedStyle(document.body).fontFamily,
      resources: performance.getEntriesByType("resource").map((entry) => entry.name),
    };
  });
  expect(fonts.check).toBe(true);
  expect(fonts.body.split(",")[0]?.replaceAll('"', "")).toBe("Geist Variable");
  const origin = new URL(world.baseURL).origin;
  expect(fonts.resources.filter((name) => new URL(name).origin !== origin)).toEqual([]);
  expect(fonts.resources.some((name) => name.endsWith(".woff2"))).toBe(true);
  await expect(page.getByRole("button", { name: "Sign out" }).locator("svg")).toHaveCount(1);
});
