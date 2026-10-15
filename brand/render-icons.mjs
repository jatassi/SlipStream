import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";

const { chromium } = createRequire(new URL("../web/package.json", import.meta.url))("playwright");

const master = readFileSync(new URL("app-icon.svg", import.meta.url), "utf8");
// iOS rounds a touch icon itself and paints transparent corners black.
const square = master.replace(' rx="14"', "");
if (square === master) {
  throw new Error('app-icon.svg has no rx="14" tile corner to square off for the touch icon');
}
const icons = [
  { name: "apple-touch-icon.png", size: 180, svg: square },
  { name: "icon-192.png", size: 192, svg: master },
  { name: "icon-512.png", size: 512, svg: master },
];

const browser = await chromium.launch();
for (const { name, size, svg } of icons) {
  const page = await browser.newPage({ viewport: { width: size, height: size } });
  await page.setContent(`<style>body{margin:0}svg{display:block;width:100vw;height:100vh}</style>${svg}`);
  await page.screenshot({
    path: fileURLToPath(new URL(`../web/public/${name}`, import.meta.url)),
    omitBackground: true,
  });
  await page.close();
}
await browser.close();
