import type { Problem } from "./problem.ts";
import type { SetAddress409Code } from "./setAddress409Code.ts";

export type SetAddress409 = Problem & {
  code?: SetAddress409Code;
};
