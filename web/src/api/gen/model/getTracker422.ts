import type { GetTracker422Code } from "./getTracker422Code.ts";
import type { Problem } from "./problem.ts";

export type GetTracker422 = Problem & {
  code?: GetTracker422Code;
};
