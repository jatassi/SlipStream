import type { GrabWaiting422Code } from "./grabWaiting422Code.ts";
import type { Problem } from "./problem.ts";

export type GrabWaiting422 = Problem & {
  code?: GrabWaiting422Code;
};
