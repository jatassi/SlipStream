import type { GetSuperseded500Code } from "./getSuperseded500Code.ts";
import type { Problem } from "./problem.ts";

export type GetSuperseded500 = Problem & {
  code?: GetSuperseded500Code;
};
