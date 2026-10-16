import type { Problem } from "./problem.ts";
import type { SetAddress401Code } from "./setAddress401Code.ts";

export type SetAddress401 = Problem & {
  code?: SetAddress401Code;
};
