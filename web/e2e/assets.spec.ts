import { z } from "zod";

import { expect, test } from "./fixtures.ts";
import { shellNav } from "./shell.ts";

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
  await expect(
    shellNav(page)
      .getByRole("link", { name: /^Activity\b/u })
      .locator("svg"),
  ).toHaveCount(1);
});

const Manifest = z.object({
  name: z.string(),
  display: z.string(),
  icons: z.array(z.object({ src: z.string(), sizes: z.string() })),
});

test("the favicon, touch icon and manifest come from the Node", async ({ page }) => {
  await page.goto("/sign-in");
  const served = (hrefs: string[]) =>
    page.evaluate(
      (urls) =>
        Promise.all(
          urls.map(async (url) => {
            const response = await fetch(url);
            return `${response.status} ${response.headers.get("content-type")}`;
          }),
        ),
      hrefs,
    );
  const links = await page.evaluate(() =>
    [...document.querySelectorAll<HTMLLinkElement>("link[rel]")]
      .filter((link) => link.rel !== "stylesheet" && link.rel !== "modulepreload")
      .map((link) => ({ rel: link.rel, href: link.href })),
  );
  expect(links.map((link) => link.rel)).toEqual(["icon", "apple-touch-icon", "manifest"]);
  expect(await served(links.map((link) => link.href))).toEqual([
    "200 image/svg+xml",
    "200 image/png",
    "200 application/manifest+json",
  ]);

  const manifestURL = links[2]?.href ?? "";
  const manifest = Manifest.parse(
    await page.evaluate(async (url): Promise<unknown> => (await fetch(url)).json(), manifestURL),
  );
  expect(manifest.name).toBe("SlipStream");
  expect(manifest.display).toBe("standalone");
  expect(manifest.icons.map((icon) => icon.sizes)).toEqual(["192x192", "512x512"]);
  expect(await served(manifest.icons.map((icon) => new URL(icon.src, manifestURL).href))).toEqual([
    "200 image/png",
    "200 image/png",
  ]);
});
