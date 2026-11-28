import type { Problem } from "./problem.ts";
import type { RerunNodeTests404Code } from "./rerunNodeTests404Code.ts";

export type RerunNodeTests404 = Problem & {
  code?: RerunNodeTests404Code;
};
