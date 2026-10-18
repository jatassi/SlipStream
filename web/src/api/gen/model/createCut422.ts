import type { CreateCut422Code } from "./createCut422Code.ts";
import type { Problem } from "./problem.ts";

export type CreateCut422 = Problem & {
  code?: CreateCut422Code;
};
