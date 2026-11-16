import type { GrabWaiting500Code } from "./grabWaiting500Code.ts";
import type { Problem } from "./problem.ts";

export type GrabWaiting500 = Problem & {
  code?: GrabWaiting500Code;
};
