import { expect, onTestFinished, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { CellPopover, cellTrigger, useCellPopover } from "./cell-popover";

type Cell = { code: string; detail: string };

const cells: Cell[] = [
  { code: "S01E01", detail: "Archive 1080p is in the library." },
  { code: "S01E02", detail: "Archive 1080p is missing." },
];

function Row() {
  const popover = useCellPopover<Cell>();
  return (
    <>
      <button type="button">Elsewhere</button>
      {cells.map((cell) => (
        <button key={cell.code} type="button" {...cellTrigger(popover.controls, cell)}>
          {cell.code}
        </button>
      ))}
      <CellPopover state={popover} label={(cell) => cell.code}>
        {(cell) => cell.detail}
      </CellPopover>
    </>
  );
}

function fakeTimers() {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  onTestFinished(() => {
    vi.useRealTimers();
  });
}

const detail = (code: string) => page.getByRole("dialog", { name: code });

test("hovering a cell opens its detail after 120ms and moving to the next cell moves it", async () => {
  renderKit(<Row />);
  fakeTimers();
  await userEvent.hover(page.getByRole("button", { name: "S01E01" }));
  await vi.advanceTimersByTimeAsync(119);
  await expect.element(detail("S01E01")).not.toBeInTheDocument();
  await vi.advanceTimersByTimeAsync(1);
  await expect.element(detail("S01E01")).toHaveTextContent("Archive 1080p is in the library.");

  await userEvent.hover(page.getByRole("button", { name: "S01E02" }));
  await expect.element(detail("S01E02")).toHaveTextContent("Archive 1080p is missing.");
  expect(page.getByRole("dialog").all()).toHaveLength(1);
});

test("a hovered detail hides 160ms after the pointer leaves", async () => {
  renderKit(<Row />);
  fakeTimers();
  await userEvent.hover(page.getByRole("button", { name: "S01E01" }));
  await vi.advanceTimersByTimeAsync(120);
  await expect.element(detail("S01E01")).toBeVisible();
  await userEvent.hover(page.getByRole("button", { name: "Elsewhere" }));
  await vi.advanceTimersByTimeAsync(159);
  await expect.element(detail("S01E01")).toBeVisible();
  await vi.advanceTimersByTimeAsync(1);
  await expect.element(detail("S01E01")).not.toBeInTheDocument();
});

test("focusing a cell opens its detail and Escape closes it, leaving focus on the cell", async () => {
  renderKit(<Row />);
  await userEvent.tab();
  await userEvent.tab();
  const cell = page.getByRole("button", { name: "S01E01" });
  await expect.element(cell).toHaveFocus();
  await expect.element(detail("S01E01")).toBeVisible();
  await userEvent.keyboard("{Escape}");
  await expect.element(detail("S01E01")).not.toBeInTheDocument();
  await expect.element(cell).toHaveFocus();
});

test("a click pins the detail, which stays when the pointer leaves and Escape closes back to the cell", async () => {
  renderKit(<Row />);
  const cell = page.getByRole("button", { name: "S01E02" });
  await cell.click();
  await expect.element(detail("S01E02")).toHaveFocus();
  await userEvent.hover(page.getByRole("button", { name: "S01E01" }));
  await new Promise((resolve) => {
    setTimeout(resolve, 400);
  });
  await expect.element(detail("S01E02")).toBeVisible();
  await expect.element(detail("S01E01")).not.toBeInTheDocument();
  await userEvent.keyboard("{Escape}");
  await expect.element(detail("S01E02")).not.toBeInTheDocument();
  await expect.element(cell).toHaveFocus();
});

test("a second click on the pinned cell or a press outside closes it", async () => {
  renderKit(<Row />);
  const cell = page.getByRole("button", { name: "S01E02" });
  await cell.click();
  await expect.element(detail("S01E02")).toBeVisible();
  await cell.click();
  await expect.element(detail("S01E02")).not.toBeInTheDocument();

  await cell.click();
  await expect.element(detail("S01E02")).toBeVisible();
  await page.getByRole("button", { name: "Elsewhere" }).click();
  await expect.element(detail("S01E02")).not.toBeInTheDocument();
});
