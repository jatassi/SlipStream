import type { GetTarget409Code } from "./getTarget409Code.ts";
import type { Problem } from "./problem.ts";

export type GetTarget409 = Problem & {
  code?: GetTarget409Code;
};
