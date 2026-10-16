import type { Problem } from "./problem.ts";
import type { UpdateOwnAccount409Code } from "./updateOwnAccount409Code.ts";

export type UpdateOwnAccount409 = Problem & {
  code?: UpdateOwnAccount409Code;
};
