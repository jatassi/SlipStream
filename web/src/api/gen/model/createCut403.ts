import type { CreateCut403Code } from "./createCut403Code.ts";
import type { Problem } from "./problem.ts";

export type CreateCut403 = Problem & {
  code?: CreateCut403Code;
};
