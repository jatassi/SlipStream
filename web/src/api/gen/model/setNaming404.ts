import type { Problem } from "./problem.ts";
import type { SetNaming404Code } from "./setNaming404Code.ts";

export type SetNaming404 = Problem & {
  code?: SetNaming404Code;
};
