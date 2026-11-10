import type { CancelGrab401Code } from "./cancelGrab401Code.ts";
import type { Problem } from "./problem.ts";

export type CancelGrab401 = Problem & {
  code?: CancelGrab401Code;
};
