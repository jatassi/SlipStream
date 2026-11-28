import type { Problem } from "./problem.ts";
import type { RerunNodeTests422Code } from "./rerunNodeTests422Code.ts";

export type RerunNodeTests422 = Problem & {
  code?: RerunNodeTests422Code;
};
