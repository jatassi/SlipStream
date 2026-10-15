import type { GetCounts403Code } from "./getCounts403Code.ts";
import type { Problem } from "./problem.ts";

export type GetCounts403 = Problem & {
  code?: GetCounts403Code;
};
