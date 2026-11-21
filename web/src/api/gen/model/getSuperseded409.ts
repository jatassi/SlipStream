import type { GetSuperseded409Code } from "./getSuperseded409Code.ts";
import type { Problem } from "./problem.ts";

export type GetSuperseded409 = Problem & {
  code?: GetSuperseded409Code;
};
