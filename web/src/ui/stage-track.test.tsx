import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";

import { StageTrack, StageWord } from "./stage-track";

test("a running stage track is a progressbar named by its stage and title", async () => {
  renderKit(
    <StageTrack
      title="Dune: Part Two"
      progress={{ kind: "running", stage: "downloading" }}
      skipped={["conversion"]}
    />,
  );
  const bar = page.getByRole("progressbar", { name: "Downloading Dune: Part Two" });
  await expect.element(bar).toHaveAttribute("aria-valuetext", "Downloading");
  await expect.element(bar).toHaveAttribute("aria-valuenow", "1");
  await expect.element(bar).toHaveAttribute("aria-valuemax", "5");
});

test("a failed or finished stage track is an image with a label", async () => {
  renderKit(
    <>
      <StageTrack title="Dune: Part Two" progress={{ kind: "failed", stage: "import" }} />
      <StageTrack title="Arrival" progress={{ kind: "available" }} />
    </>,
  );
  await expect
    .element(page.getByRole("img", { name: "Dune: Part Two failed at Import" }))
    .toBeVisible();
  await expect.element(page.getByRole("img", { name: "Arrival is Available" })).toBeVisible();
  await expect.element(page.getByRole("progressbar")).not.toBeInTheDocument();
});

test("a track names its ends only where it asks", async () => {
  const plain = renderKit(<StageTrack title="Arrival" progress={{ kind: "available" }} />);
  const ended = renderKit(
    <StageTrack title="Dune" progress={{ kind: "failed", stage: "import" }} labels="ends" />,
  );
  await expect.element(page.getByRole("img", { name: "Dune failed at Import" })).toBeVisible();
  expect([plain.textContent, ended.textContent]).toEqual(["", "SearchingAvailable"]);
});

test("on a phone a track names its ends unasked", async () => {
  const container = renderKit(<StageTrack title="Arrival" progress={{ kind: "available" }} />, {
    shell: "phone",
  });
  await expect.element(page.getByRole("img", { name: "Arrival is Available" })).toBeVisible();
  expect(container.textContent).toBe("SearchingAvailable");
});

test("the stage word opens the track's legend, each stop with its word", async () => {
  renderKit(
    <StageWord
      word="Downloading"
      title="Dune"
      progress={{ kind: "running", stage: "downloading" }}
      skipped={["conversion"]}
    />,
  );
  await userEvent.click(page.getByRole("button", { name: "Downloading" }));
  const legend = page.getByRole("dialog", { name: "Stages of Dune" });
  await expect
    .poll(() =>
      legend
        .getByRole("listitem")
        .all()
        .map((stop) => stop.element().textContent),
    )
    .toEqual([
      "SearchingDone",
      "DownloadingIn progress",
      "TransferNot started",
      "ImportNot started",
      "ConversionNot needed",
      "AvailableNot started",
    ]);
});
