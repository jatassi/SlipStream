import type { Problem } from "./problem.ts";
import type { SetWorkRoot404Code } from "./setWorkRoot404Code.ts";

export type SetWorkRoot404 = Problem & {
  code?: SetWorkRoot404Code;
};
