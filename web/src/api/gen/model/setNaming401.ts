import type { Problem } from "./problem.ts";
import type { SetNaming401Code } from "./setNaming401Code.ts";

export type SetNaming401 = Problem & {
  code?: SetNaming401Code;
};
