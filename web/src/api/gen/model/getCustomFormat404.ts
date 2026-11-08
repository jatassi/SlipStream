import type { GetCustomFormat404Code } from "./getCustomFormat404Code.ts";
import type { Problem } from "./problem.ts";

export type GetCustomFormat404 = Problem & {
  code?: GetCustomFormat404Code;
};
