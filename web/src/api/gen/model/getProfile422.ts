import type { GetProfile422Code } from "./getProfile422Code.ts";
import type { Problem } from "./problem.ts";

export type GetProfile422 = Problem & {
  code?: GetProfile422Code;
};
