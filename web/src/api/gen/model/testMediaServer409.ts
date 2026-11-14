import type { Problem } from "./problem.ts";
import type { TestMediaServer409Code } from "./testMediaServer409Code.ts";

export type TestMediaServer409 = Problem & {
  code?: TestMediaServer409Code;
};
