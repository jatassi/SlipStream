import type { GetServing500Code } from "./getServing500Code.ts";
import type { Problem } from "./problem.ts";

export type GetServing500 = Problem & {
  code?: GetServing500Code;
};
