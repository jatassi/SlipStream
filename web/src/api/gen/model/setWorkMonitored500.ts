import type { Problem } from "./problem.ts";
import type { SetWorkMonitored500Code } from "./setWorkMonitored500Code.ts";

export type SetWorkMonitored500 = Problem & {
  code?: SetWorkMonitored500Code;
};
