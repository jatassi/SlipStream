import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { calendarDayWords, isoDay, monthSpan } from "../format/days";
import { renderKit } from "../test/render-kit";
import { MonthGrid } from "./month-grid";

test("the grid draws six weeks from the week's first day, with the weekday heads and Today marked", async () => {
  const month = { year: 2026, month: 11 };
  const { from } = monthSpan(month, 1);
  renderKit(
    <MonthGrid
      label="November 2026"
      from={from}
      month={month}
      today={isoDay("2026-11-17")}
      renderDay={(day) => <span>{day === "2026-11-20" ? "Harbor Lights" : null}</span>}
    />,
  );
  const grid = page.getByRole("table", { name: "November 2026" });
  await expect.element(grid.getByRole("columnheader").first()).toHaveAttribute("abbr", "Monday");
  expect(grid.getByRole("cell").elements()).toHaveLength(42);
  await expect
    .element(grid.getByRole("cell").first())
    .toMatchTextContent(calendarDayWords(isoDay("2026-10-26")));
  await expect
    .element(grid.getByRole("cell").filter({ hasText: calendarDayWords(isoDay("2026-11-17")) }))
    .toHaveAttribute("aria-current", "date");
  await expect
    .element(grid.getByRole("cell").filter({ hasText: "Harbor Lights" }))
    .toMatchTextContent(calendarDayWords(isoDay("2026-11-20")));
});
