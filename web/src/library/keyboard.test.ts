import { expect, test } from "vitest";

import { gridRows } from "./grid-layout";
import { gridMove, tableMove } from "./keyboard";

test("arrow keys move one Work across and keep the column up and down, past section headings", () => {
  const order = ["a", "b", "c", "d", "e", "f", "g"];
  const layout = gridRows(
    {
      order,
      sections: [
        { section: { key: "A", label: "A" }, start: 0, count: 5 },
        { section: { key: "B", label: "B" }, start: 5, count: 2 },
      ],
    },
    3,
  );
  expect(
    layout.rows.map((row) => (row.kind === "section" ? row.section.label : row.ids.join(""))),
  ).toEqual(["A", "abc", "de", "B", "fg"]);
  const count = order.length;
  expect(gridMove(layout, 2, "down", count)).toBe(4);
  expect(gridMove(layout, 4, "down", count)).toBe(6);
  expect(gridMove(layout, 3, "down", count)).toBe(5);
  expect(gridMove(layout, 5, "up", count)).toBe(3);
  expect(gridMove(layout, 1, "up", count)).toBe(1);
  expect(gridMove(layout, 2, "right", count)).toBe(3);
  expect(gridMove(layout, 0, "left", count)).toBe(0);
  expect(gridMove(layout, 3, "last", count)).toBe(6);
  expect(tableMove(0, "down", count)).toBe(1);
  expect(tableMove(6, "down", count)).toBe(6);
  expect(tableMove(3, "first", count)).toBe(0);
  expect(tableMove(3, "left", count)).toBe(3);
});
