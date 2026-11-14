import type { GrabWaiting501Code } from "./grabWaiting501Code.ts";
import type { Problem } from "./problem.ts";

export type GrabWaiting501 = Problem & {
  code?: GrabWaiting501Code;
};
