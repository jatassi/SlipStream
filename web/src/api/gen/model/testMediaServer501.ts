import type { Problem } from "./problem.ts";
import type { TestMediaServer501Code } from "./testMediaServer501Code.ts";

export type TestMediaServer501 = Problem & {
  code?: TestMediaServer501Code;
};
