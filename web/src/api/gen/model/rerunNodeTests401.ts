import type { Problem } from "./problem.ts";
import type { RerunNodeTests401Code } from "./rerunNodeTests401Code.ts";

export type RerunNodeTests401 = Problem & {
  code?: RerunNodeTests401Code;
};
