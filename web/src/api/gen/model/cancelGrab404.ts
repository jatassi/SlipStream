import type { CancelGrab404Code } from "./cancelGrab404Code.ts";
import type { Problem } from "./problem.ts";

export type CancelGrab404 = Problem & {
  code?: CancelGrab404Code;
};
