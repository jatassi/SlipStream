import type { CancelGrab403Code } from "./cancelGrab403Code.ts";
import type { Problem } from "./problem.ts";

export type CancelGrab403 = Problem & {
  code?: CancelGrab403Code;
};
