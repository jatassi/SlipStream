import type { Problem } from "./problem.ts";
import type { SetFirstRunStep401Code } from "./setFirstRunStep401Code.ts";

export type SetFirstRunStep401 = Problem & {
  code?: SetFirstRunStep401Code;
};
