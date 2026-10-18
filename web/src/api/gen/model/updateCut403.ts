import type { Problem } from "./problem.ts";
import type { UpdateCut403Code } from "./updateCut403Code.ts";

export type UpdateCut403 = Problem & {
  code?: UpdateCut403Code;
};
