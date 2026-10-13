import { expect, test } from "vitest";

import vectors from "../../../internal/identity/testdata/setup_code_vectors.json";
import { formatSetupCode, normalizeSetupCode } from "./setup-code";

test.each(vectors)("$input normalises to $normalized and formats as $formatted", (vector) => {
  expect(normalizeSetupCode(vector.input)).toBe(vector.normalized);
  expect(formatSetupCode(vector.normalized)).toBe(vector.formatted);
});

test("the vectors carry the cases both parsers must agree on", () => {
  expect(vectors.map(({ input }) => input)).toEqual([
    "7k2m q9xd 4hbt",
    "7K2M-Q9XD-4HBT",
    "7K2MQ9XD4HBT",
    "OIL0",
    "7k2m-q9xd-4hbtx",
    "7K2MQ9",
  ]);
});

test("a long code keeps every character", () => {
  expect(normalizeSetupCode("7k2m-q9xd-4hbtx")).toHaveLength(13);
});
