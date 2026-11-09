import type { Problem } from "./problem.ts";
import type { SetNaming500Code } from "./setNaming500Code.ts";

export type SetNaming500 = Problem & {
  code?: SetNaming500Code;
};
