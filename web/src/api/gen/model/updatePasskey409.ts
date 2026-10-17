import type { Problem } from "./problem.ts";
import type { UpdatePasskey409Code } from "./updatePasskey409Code.ts";

export type UpdatePasskey409 = Problem & {
  code?: UpdatePasskey409Code;
};
