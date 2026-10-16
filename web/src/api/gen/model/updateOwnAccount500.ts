import type { Problem } from "./problem.ts";
import type { UpdateOwnAccount500Code } from "./updateOwnAccount500Code.ts";

export type UpdateOwnAccount500 = Problem & {
  code?: UpdateOwnAccount500Code;
};
