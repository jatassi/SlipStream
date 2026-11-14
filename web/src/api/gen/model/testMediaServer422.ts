import type { Problem } from "./problem.ts";
import type { TestMediaServer422Code } from "./testMediaServer422Code.ts";

export type TestMediaServer422 = Problem & {
  code?: TestMediaServer422Code;
};
