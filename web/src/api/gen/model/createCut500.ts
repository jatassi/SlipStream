import type { CreateCut500Code } from "./createCut500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateCut500 = Problem & {
  code?: CreateCut500Code;
};
