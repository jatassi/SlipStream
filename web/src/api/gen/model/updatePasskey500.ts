import type { Problem } from "./problem.ts";
import type { UpdatePasskey500Code } from "./updatePasskey500Code.ts";

export type UpdatePasskey500 = Problem & {
  code?: UpdatePasskey500Code;
};
