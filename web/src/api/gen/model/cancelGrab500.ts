import type { CancelGrab500Code } from "./cancelGrab500Code.ts";
import type { Problem } from "./problem.ts";

export type CancelGrab500 = Problem & {
  code?: CancelGrab500Code;
};
