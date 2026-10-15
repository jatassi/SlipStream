import { expect, test } from "vitest";

import { matches } from "./match";

test("a query matches a substring in any case", () => {
  expect(matches("dun", "Dune: Part Two")).toBe(true);
  expect(matches("PART t", "Dune: Part Two")).toBe(true);
  expect(matches("dunes", "Dune: Part Two")).toBe(false);
});

test("accents count as their plain letters on either side", () => {
  expect(matches("amelie", "Amélie")).toBe(true);
  expect(matches("AMÉLIE", "amelie")).toBe(true);
  expect(matches("sao", "São Paulo")).toBe(true);
  expect(matches("ämelie", "Amelia")).toBe(false);
});

test("any of the texts can match", () => {
  expect(matches("quiet", "The Garden", "Quiet Orchard")).toBe(true);
  expect(matches("quiet", "The Garden", "Loud Orchard")).toBe(false);
});
