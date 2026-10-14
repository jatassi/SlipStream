import type { GetWorkCells404Code } from "./getWorkCells404Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkCells404 = Problem & {
  code?: GetWorkCells404Code;
};
