import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { SegmentCount } from "./segmented";
import { SegmentedLinks, segmentedLink } from "./segmented-links";

function TypeSwitch() {
  return (
    <SegmentedLinks label="Content Types">
      <a href="/library" className={segmentedLink}>
        All
        <SegmentCount count={120} />
      </a>
      <a href="/library/cinema" className={segmentedLink} aria-current="page">
        Cinema
        <SegmentCount count={80} />
      </a>
      <a href="/library/anthology" className={segmentedLink}>
        Anthology
      </a>
    </SegmentedLinks>
  );
}

test("segmented links are links in a named nav, with the current one selected", async () => {
  renderKit(<TypeSwitch />);
  const nav = page.getByRole("navigation", { name: "Content Types" });
  await expect.element(nav).toBeVisible();
  const current = nav.getByRole("link", { name: "Cinema 80" });
  await expect.element(current).toHaveAttribute("href", "/library/cinema");
  await expect.element(current).toHaveAttribute("aria-current", "page");
  await expect.element(nav.getByRole("link", { name: "All 120" })).toBeVisible();
  await expect.element(nav.getByRole("link", { name: "Anthology" })).toBeVisible();
});
