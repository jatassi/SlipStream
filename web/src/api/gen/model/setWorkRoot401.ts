import type { Problem } from "./problem.ts";
import type { SetWorkRoot401Code } from "./setWorkRoot401Code.ts";

export type SetWorkRoot401 = Problem & {
  code?: SetWorkRoot401Code;
};
