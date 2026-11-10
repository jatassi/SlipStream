import type { CancelGrab501Code } from "./cancelGrab501Code.ts";
import type { Problem } from "./problem.ts";

export type CancelGrab501 = Problem & {
  code?: CancelGrab501Code;
};
