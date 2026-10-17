import type { Problem } from "./problem.ts";
import type { WriteOff401Code } from "./writeOff401Code.ts";

export type WriteOff401 = Problem & {
  code?: WriteOff401Code;
};
