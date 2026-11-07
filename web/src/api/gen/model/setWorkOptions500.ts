import type { Problem } from "./problem.ts";
import type { SetWorkOptions500Code } from "./setWorkOptions500Code.ts";

export type SetWorkOptions500 = Problem & {
  code?: SetWorkOptions500Code;
};
