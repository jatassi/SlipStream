import type { GrabWaiting404Code } from "./grabWaiting404Code.ts";
import type { Problem } from "./problem.ts";

export type GrabWaiting404 = Problem & {
  code?: GrabWaiting404Code;
};
