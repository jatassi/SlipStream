import type { Problem } from "./problem.ts";
import type { SetAddress500Code } from "./setAddress500Code.ts";

export type SetAddress500 = Problem & {
  code?: SetAddress500Code;
};
