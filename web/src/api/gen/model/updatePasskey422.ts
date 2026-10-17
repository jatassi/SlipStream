import type { Problem } from "./problem.ts";
import type { UpdatePasskey422Code } from "./updatePasskey422Code.ts";

export type UpdatePasskey422 = Problem & {
  code?: UpdatePasskey422Code;
};
