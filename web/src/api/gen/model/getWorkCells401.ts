import type { GetWorkCells401Code } from "./getWorkCells401Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkCells401 = Problem & {
  code?: GetWorkCells401Code;
};
