import type { Problem } from "./problem.ts";
import type { UpdateOwnAccount403Code } from "./updateOwnAccount403Code.ts";

export type UpdateOwnAccount403 = Problem & {
  code?: UpdateOwnAccount403Code;
};
