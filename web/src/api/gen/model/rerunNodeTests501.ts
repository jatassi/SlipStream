import type { Problem } from "./problem.ts";
import type { RerunNodeTests501Code } from "./rerunNodeTests501Code.ts";

export type RerunNodeTests501 = Problem & {
  code?: RerunNodeTests501Code;
};
