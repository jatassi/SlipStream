import type { Problem } from "./problem.ts";
import type { UpdateHomeSettings403Code } from "./updateHomeSettings403Code.ts";

export type UpdateHomeSettings403 = Problem & {
  code?: UpdateHomeSettings403Code;
};
