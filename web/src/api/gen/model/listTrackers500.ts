import type { ListTrackers500Code } from "./listTrackers500Code.ts";
import type { Problem } from "./problem.ts";

export type ListTrackers500 = Problem & {
  code?: ListTrackers500Code;
};
