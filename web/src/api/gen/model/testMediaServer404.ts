import type { Problem } from "./problem.ts";
import type { TestMediaServer404Code } from "./testMediaServer404Code.ts";

export type TestMediaServer404 = Problem & {
  code?: TestMediaServer404Code;
};
