import { expect, test } from "@playwright/test";

import { untilStill } from "./still.ts";

// The fill takes a new value every half second and eases for a second to
// each, so it is never at rest, as a downloading torrent's bar is.
const liveBar = `
  <div role="progressbar" aria-label="Downloads">
    <div id="fill" style="height: 4px; width: 0; background: black; transition: width 1s linear"></div>
  </div>
  <script>
    let value = 0;
    setInterval(() => {
      value = (value + 7) % 100;
      document.getElementById("fill").style.width = value + "%";
    }, 500);
  </script>`;

test("a wait for stillness passes over a live progress bar", async ({ page }) => {
  await page.setContent(liveBar);
  await expect
    .poll(() => page.evaluate(() => document.getAnimations().length), { message: "the bar moves" })
    .toBe(1);
  await untilStill(page);
});
