import type { GetSuperseded401Code } from "./getSuperseded401Code.ts";
import type { Problem } from "./problem.ts";

export type GetSuperseded401 = Problem & {
  code?: GetSuperseded401Code;
};
