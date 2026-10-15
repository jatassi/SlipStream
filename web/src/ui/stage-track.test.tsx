import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { StageTrack } from "./stage-track";

const shells = [
  { shell: "desktop", height: "6px" },
  { shell: "phone", height: "8px" },
] as const satisfies readonly { shell: ShellKind; height: string }[];

describe.each(shells)("in the $shell shell", ({ shell, height }) => {
  test("a running stage track is a progressbar named by its stage and title", async () => {
    renderKit(
      <StageTrack
        title="Dune: Part Two"
        progress={{ kind: "running", stage: "downloading" }}
        skipped={["conversion"]}
      />,
      { shell },
    );
    const bar = page.getByRole("progressbar", { name: "Downloading Dune: Part Two" });
    await expect.element(bar).toHaveAttribute("aria-valuetext", "Downloading");
    await expect.element(bar).toHaveAttribute("aria-valuenow", "1");
    await expect.element(bar).toHaveAttribute("aria-valuemax", "5");
    expect(getComputedStyle(bar.element()).height).toBe(height);
  });

  test("a failed or finished stage track is an image with a label", async () => {
    renderKit(
      <>
        <StageTrack title="Dune: Part Two" progress={{ kind: "failed", stage: "import" }} />
        <StageTrack title="Arrival" progress={{ kind: "available" }} />
      </>,
      { shell },
    );
    await expect
      .element(page.getByRole("img", { name: "Dune: Part Two failed at Import" }))
      .toBeVisible();
    await expect.element(page.getByRole("img", { name: "Arrival is Available" })).toBeVisible();
    await expect.element(page.getByRole("progressbar")).not.toBeInTheDocument();
  });
});

test("the Inspector's track is 8px in the desktop shell", async () => {
  renderKit(<StageTrack title="Arrival" progress={{ kind: "available" }} size="large" />);
  const track = page.getByRole("img", { name: "Arrival is Available" });
  await expect.element(track).toBeVisible();
  expect(getComputedStyle(track.element()).height).toBe("8px");
});
