import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { SegmentCount } from "./segmented";
import { SegmentedLinks, segmentedLink } from "./segmented-links";
import type { ShellKind } from "./shell-context";

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
  const other = nav.getByRole("link", { name: "All 120" });
  await expect.element(current).toHaveAttribute("href", "/library/cinema");
  await expect.element(current).toHaveAttribute("aria-current", "page");
  await expect.element(nav.getByRole("link", { name: "Anthology" })).toBeVisible();
  expect(getComputedStyle(current.element()).backgroundColor).not.toBe(
    getComputedStyle(other.element()).backgroundColor,
  );
  expect(getComputedStyle(current.element()).height).toBe("28px");
});

test.each([
  ["desktop", "28px"],
  ["phone", "44px"],
] satisfies [ShellKind, string][])(
  "in the %s shell a segment link is %s tall",
  async (shell, height) => {
    renderKit(<TypeSwitch />, { shell });
    const link = page.getByRole("link", { name: "Anthology" });
    await expect.element(link).toBeVisible();
    expect(getComputedStyle(link.element()).height).toBe(height);
  },
);

test("on phone the links span their container", async () => {
  renderKit(
    <div style={{ width: "360px" }}>
      <TypeSwitch />
    </div>,
    { shell: "phone" },
  );
  const nav = page.getByRole("navigation", { name: "Content Types" });
  await expect.element(nav).toBeVisible();
  expect(getComputedStyle(nav.element()).width).toBe("360px");
});
