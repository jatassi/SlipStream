import type { CreateCut401Code } from "./createCut401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateCut401 = Problem & {
  code?: CreateCut401Code;
};
