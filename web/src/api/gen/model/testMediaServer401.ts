import type { Problem } from "./problem.ts";
import type { TestMediaServer401Code } from "./testMediaServer401Code.ts";

export type TestMediaServer401 = Problem & {
  code?: TestMediaServer401Code;
};
