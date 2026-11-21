import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import {
  EpisodeStrip,
  type Slot,
  byCounts,
  inOrder,
  segmentsOf,
  stripWords,
} from "./episode-strip";

const repeat = (slot: Slot, times: number): Slot[] => Array.from({ length: times }, () => slot);

const shape = (slots: readonly Slot[]) => {
  const { by, segments } = segmentsOf(inOrder(slots), "header");
  return { by, segments: segments.map(({ slot, span }) => `${span} ${slot}`) };
};

test("a season that fits draws one segment per episode in order", () => {
  const season: Slot[] = ["met", "met", "missing", "quiet", "downloading"];
  expect(shape(season)).toEqual({
    by: "episode",
    segments: ["1 met", "1 met", "1 missing", "1 quiet", "1 downloading"],
  });
});

test("a long season merges runs of one state, and counts them once the runs don't fit", () => {
  expect(shape([...repeat("met", 40), ...repeat("missing", 20)])).toEqual({
    by: "run",
    segments: ["40 met", "20 missing"],
  });
  const alternating = Array.from({ length: 60 }, (_, index): Slot =>
    index % 2 === 0 ? "missing" : "met",
  );
  expect(shape(alternating)).toEqual({ by: "count", segments: ["30 met", "30 missing"] });
});

test("a rollup's counts run in progress order, so Met grows from the left", () => {
  const strip = segmentsOf(byCounts({ failed: 1, met: 6, missing: 3 }), "poster");
  expect(strip.segments.map(({ slot }) => slot)).toEqual([
    ...repeat("met", 6),
    ...repeat("missing", 3),
    "failed",
  ]);
  expect(stripWords(strip)).toBe("1 Failed, 3 Missing, 6 Met");
});

test("a named strip is an image of its counts in the state order, sized by them", async () => {
  const slots: Slot[] = [...repeat("met", 6), ...repeat("unreleased", 2), "failed", "quiet"];
  renderKit(<EpisodeStrip strip={segmentsOf(inOrder(slots), "header")} look="header" named />);
  const strip = page.getByRole("img", { name: "1 Failed, 2 Unreleased, 6 Met, 1 Not monitored" });
  await expect.element(strip).toBeVisible();
  const box = strip.element().getBoundingClientRect();
  expect([box.width, box.height]).toEqual([180, 8]);
  expect(strip.element().children).toHaveLength(10);
});

test("an empty strip says so, and one the words beside it name stays silent", async () => {
  renderKit(
    <>
      <EpisodeStrip strip={segmentsOf([], "header")} look="header" named />
      <EpisodeStrip strip={segmentsOf(inOrder(["met"]), "edge")} look="edge" named={false} />
    </>,
  );
  await expect.element(page.getByRole("img", { name: "Nothing to download" })).toBeVisible();
  expect(document.querySelectorAll("[aria-hidden=true]")).toHaveLength(1);
});
