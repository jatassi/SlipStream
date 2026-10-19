import type { GetHealthSettings500Code } from "./getHealthSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetHealthSettings500 = Problem & {
  code?: GetHealthSettings500Code;
};
