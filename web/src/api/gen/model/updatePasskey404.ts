import type { Problem } from "./problem.ts";
import type { UpdatePasskey404Code } from "./updatePasskey404Code.ts";

export type UpdatePasskey404 = Problem & {
  code?: UpdatePasskey404Code;
};
