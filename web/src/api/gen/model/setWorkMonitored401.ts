import type { Problem } from "./problem.ts";
import type { SetWorkMonitored401Code } from "./setWorkMonitored401Code.ts";

export type SetWorkMonitored401 = Problem & {
  code?: SetWorkMonitored401Code;
};
