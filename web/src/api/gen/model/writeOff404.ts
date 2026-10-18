import type { Problem } from "./problem.ts";
import type { WriteOff404Code } from "./writeOff404Code.ts";

export type WriteOff404 = Problem & {
  code?: WriteOff404Code;
};
