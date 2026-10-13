import type { GetWorkCells500Code } from "./getWorkCells500Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkCells500 = Problem & {
  code?: GetWorkCells500Code;
};
