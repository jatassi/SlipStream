import type { Problem } from "./problem.ts";
import type { TestMediaServer500Code } from "./testMediaServer500Code.ts";

export type TestMediaServer500 = Problem & {
  code?: TestMediaServer500Code;
};
