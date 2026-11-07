import type { GetTracker403Code } from "./getTracker403Code.ts";
import type { Problem } from "./problem.ts";

export type GetTracker403 = Problem & {
  code?: GetTracker403Code;
};
