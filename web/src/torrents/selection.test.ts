import { expect, test } from "vitest";

import { noPicks, pickedOf, toggle, toggleAll } from "./selection";

const order = ["a", "b", "c", "d", "e"];
const ids = (picks: { ids: ReadonlySet<string> }) => [...picks.ids].toSorted();

test("a toggle picks a row and a second toggle drops it", () => {
  const one = toggle(noPicks, "b", order, false);
  expect(ids(one)).toEqual(["b"]);
  expect(ids(toggle(one, "b", order, false))).toEqual([]);
});

test("a shift toggle picks every row from the last toggled one, either way", () => {
  const down = toggle(toggle(noPicks, "b", order, false), "d", order, true);
  expect(ids(down)).toEqual(["b", "c", "d"]);
  const up = toggle(toggle(noPicks, "d", order, false), "a", order, true);
  expect(ids(up)).toEqual(["a", "b", "c", "d"]);
});

test("a shift toggle on a picked row drops the range", () => {
  const all = toggleAll(noPicks, order);
  const dropped = toggle(toggle(all, "e", order, false), "c", order, true);
  expect(ids(dropped)).toEqual(["a", "b"]);
});

test("a shift toggle with no earlier toggle picks one row", () => {
  expect(ids(toggle(noPicks, "c", order, true))).toEqual(["c"]);
});

test("toggling all picks every shown row, and again picks none", () => {
  const all = toggleAll(toggle(noPicks, "a", order, false), order);
  expect(ids(all)).toEqual(order);
  expect(ids(toggleAll(all, order))).toEqual([]);
});

test("only shown rows count as picked", () => {
  const picks = toggleAll(noPicks, order);
  expect(pickedOf(picks, [{ id: "c" }, { id: "z" }])).toEqual([{ id: "c" }]);
});
