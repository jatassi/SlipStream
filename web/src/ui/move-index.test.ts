import { expect, test } from "vitest";

import { type Move, moveIndex } from "./move-index";

const cases: readonly (readonly [
  from: number,
  to: Move,
  count: number,
  page: number,
  want: number,
])[] = [
  [0, "next", 10, 4, 1],
  [9, "next", 10, 4, 9],
  [-1, "next", 10, 4, 0],
  [5, "prev", 10, 4, 4],
  [0, "prev", 10, 4, 0],
  [-1, "prev", 10, 4, 9],
  [5, "first", 10, 4, 0],
  [-1, "first", 10, 4, 0],
  [5, "last", 10, 4, 9],
  [-1, "last", 10, 4, 9],
  [2, "pageDown", 10, 4, 6],
  [8, "pageDown", 10, 4, 9],
  [-1, "pageDown", 10, 4, 3],
  [6, "pageUp", 10, 4, 2],
  [2, "pageUp", 10, 4, 0],
  [-1, "pageUp", 10, 4, 6],
  [3, "pageDown", 10, 0, 4],
  [12, "next", 10, 4, 9],
  [12, "prev", 10, 4, 9],
  [0, "next", 0, 4, -1],
  [-1, "last", 0, 4, -1],
  [0, "pageDown", 1, 4, 0],
];

test.each(cases)(
  "from %i, %s through %i rows (page %i) lands on %i",
  (from, to, count, page, want) => {
    expect(moveIndex(from, to, count, page)).toBe(want);
  },
);
