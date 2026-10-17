import type { GetAddress409Code } from "./getAddress409Code.ts";
import type { Problem } from "./problem.ts";

export type GetAddress409 = Problem & {
  code?: GetAddress409Code;
};
