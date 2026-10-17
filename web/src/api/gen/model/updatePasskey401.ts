import type { Problem } from "./problem.ts";
import type { UpdatePasskey401Code } from "./updatePasskey401Code.ts";

export type UpdatePasskey401 = Problem & {
  code?: UpdatePasskey401Code;
};
