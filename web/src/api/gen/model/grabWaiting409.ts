import type { GrabWaiting409Code } from "./grabWaiting409Code.ts";
import type { Problem } from "./problem.ts";

export type GrabWaiting409 = Problem & {
  code?: GrabWaiting409Code;
};
