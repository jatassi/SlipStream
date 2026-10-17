import type { Problem } from "./problem.ts";
import type { WriteOff422Code } from "./writeOff422Code.ts";

export type WriteOff422 = Problem & {
  code?: WriteOff422Code;
};
