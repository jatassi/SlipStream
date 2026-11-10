import type { CancelGrab409Code } from "./cancelGrab409Code.ts";
import type { Problem } from "./problem.ts";

export type CancelGrab409 = Problem & {
  code?: CancelGrab409Code;
};
