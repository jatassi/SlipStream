import type { Problem } from "./problem.ts";
import type { SetWorkRoot500Code } from "./setWorkRoot500Code.ts";

export type SetWorkRoot500 = Problem & {
  code?: SetWorkRoot500Code;
};
