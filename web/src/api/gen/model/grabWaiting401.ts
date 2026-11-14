import type { GrabWaiting401Code } from "./grabWaiting401Code.ts";
import type { Problem } from "./problem.ts";

export type GrabWaiting401 = Problem & {
  code?: GrabWaiting401Code;
};
