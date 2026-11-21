import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { ProgressBar, ProgressCell, percentOf } from "./progress";
import type { ShellKind } from "./shell-context";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a bar is a named progressbar on a 6px track, live or settled",
  async (shell) => {
    renderKit(
      <>
        <ProgressBar label="Downloading Dune" value={42} motion="live" />
        <ProgressBar label="Paused Dune" value={42} motion="settled" />
      </>,
      { shell },
    );
    for (const [name, motion] of [
      ["Downloading Dune", "live"],
      ["Paused Dune", "settled"],
    ] as const) {
      const bar = page.getByRole("progressbar", { name });
      // oxlint-disable-next-line no-await-in-loop -- one bar at a time names the one that fails
      await expect.element(bar).toHaveAttribute("aria-valuenow", "42");
      expect(bar.element()).toHaveAttribute("data-motion", motion);
      expect(bar.element().textContent).toBe("");
      const track = bar.element().firstElementChild;
      expect(track === null ? null : getComputedStyle(track).height).toBe("6px");
    }
  },
);

test("a cell reads its amount, the percent unless named, then its note", async () => {
  renderKit(
    <>
      <ProgressCell label="Downloading Dune" value={42} motion="live" note="2 min left" />
      <ProgressCell
        label="Backlog search progress"
        value={25}
        motion="live"
        amount="3 of 12 searches"
        note="home"
      />
    </>,
  );
  await expect
    .element(page.getByRole("progressbar", { name: "Downloading Dune" }))
    .toBeInTheDocument();
  const cells = [...document.querySelectorAll("[role=progressbar]")].map(
    (bar) => bar.parentElement?.textContent,
  );
  expect(cells).toEqual(["42%2 min left", "3 of 12 searcheshome"]);
});

test("a percent is floored, and an empty total reads as the caller says", () => {
  expect([
    percentOf(999, 1000, 0),
    percentOf(1000, 1000, 0),
    percentOf(0, 0, 0),
    percentOf(0, 0, 100),
  ]).toEqual([99, 100, 0, 100]);
});
