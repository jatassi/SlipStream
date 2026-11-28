import type { Problem } from "./problem.ts";
import type { RerunNodeTests403Code } from "./rerunNodeTests403Code.ts";

export type RerunNodeTests403 = Problem & {
  code?: RerunNodeTests403Code;
};
