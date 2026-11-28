import type { Problem } from "./problem.ts";
import type { RerunNodeTests500Code } from "./rerunNodeTests500Code.ts";

export type RerunNodeTests500 = Problem & {
  code?: RerunNodeTests500Code;
};
