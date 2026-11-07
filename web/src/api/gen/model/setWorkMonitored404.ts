import type { Problem } from "./problem.ts";
import type { SetWorkMonitored404Code } from "./setWorkMonitored404Code.ts";

export type SetWorkMonitored404 = Problem & {
  code?: SetWorkMonitored404Code;
};
