import type { Problem } from "./problem.ts";
import type { WriteOff403Code } from "./writeOff403Code.ts";

export type WriteOff403 = Problem & {
  code?: WriteOff403Code;
};
