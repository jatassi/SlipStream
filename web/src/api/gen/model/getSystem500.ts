import type { GetSystem500Code } from "./getSystem500Code.ts";
import type { Problem } from "./problem.ts";

export type GetSystem500 = Problem & {
  code?: GetSystem500Code;
};
