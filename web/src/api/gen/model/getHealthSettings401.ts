import type { GetHealthSettings401Code } from "./getHealthSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetHealthSettings401 = Problem & {
  code?: GetHealthSettings401Code;
};
