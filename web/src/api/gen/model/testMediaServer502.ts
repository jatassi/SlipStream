import type { Problem } from "./problem.ts";
import type { TestMediaServer502Code } from "./testMediaServer502Code.ts";

export type TestMediaServer502 = Problem & {
  code?: TestMediaServer502Code;
};
