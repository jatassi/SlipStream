import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { stateOrder as states, statusMarks } from "./state-marks";
import { StateBar, StateButton } from "./state-square";

function squareOf(name: string) {
  const square = page.getByRole("button", { name }).element().firstElementChild;
  if (square === null) {
    throw new Error(`${name} has no square`);
  }
  return square;
}

test("a cell is a 42 × 30 button around a 16px square, named with its state", async () => {
  renderKit(<StateButton state="missing" label="S01E02 Archive" />);
  const cell = page.getByRole("button", { name: "S01E02 Archive, Missing" });
  await expect.element(cell).toBeVisible();
  const box = cell.element().getBoundingClientRect();
  expect([box.width, box.height]).toEqual([42, 30]);
  const square = squareOf("S01E02 Archive, Missing").getBoundingClientRect();
  expect([square.width, square.height]).toEqual([16, 16]);
});

test("each state has its own look", async () => {
  renderKit(
    <>
      {states.map((state) => (
        <StateButton key={state} state={state} label={state} />
      ))}
    </>,
  );
  await expect.element(page.getByRole("button", { name: "met, Met" })).toBeVisible();
  const looks = states.map((state) => {
    const square = squareOf(`${state}, ${statusMarks[state].word}`);
    const style = getComputedStyle(square);
    const dot = getComputedStyle(square, "::after");
    return [style.backgroundColor, style.backgroundImage, style.boxShadow, dot.content].join("|");
  });
  expect(new Set(looks).size).toBe(states.length);
});

test("a quiet square is dimmed", async () => {
  renderKit(
    <>
      <StateButton state="met" label="Watched" />
      <StateButton state="met" label="Quiet" quiet />
    </>,
  );
  await expect.element(page.getByRole("button", { name: "Quiet, Met" })).toBeVisible();
  expect(getComputedStyle(squareOf("Watched, Met")).opacity).toBe("1");
  expect(getComputedStyle(squareOf("Quiet, Met")).opacity).toBe("0.55");
});

test("a state bar is 180 × 8, names its counts in the state order, and sizes each segment by its count", async () => {
  renderKit(<StateBar counts={{ met: 6, unreleased: 2, missing: 3, failed: 1 }} />);
  const bar = page.getByRole("img", { name: "1 Failed, 3 Missing, 2 Unreleased, 6 Met" });
  await expect.element(bar).toBeVisible();
  const box = bar.element().getBoundingClientRect();
  expect([box.width, box.height]).toEqual([180, 8]);
  const widths = [...bar.element().children].map(
    (segment) => segment.getBoundingClientRect().width,
  );
  expect(widths).toHaveLength(4);
  const [failed = 0, missing = 0, , met = 0] = widths;
  expect(missing / failed).toBeCloseTo(3, 1);
  expect(met / failed).toBeCloseTo(6, 1);
});

test("an empty state bar says so", async () => {
  renderKit(<StateBar counts={{}} />);
  await expect.element(page.getByRole("img", { name: "No cells" })).toBeVisible();
});
