import type { Problem } from "./problem.ts";
import type { WriteOff500Code } from "./writeOff500Code.ts";

export type WriteOff500 = Problem & {
  code?: WriteOff500Code;
};
