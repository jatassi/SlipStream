import type { GetAccount409Code } from "./getAccount409Code.ts";
import type { Problem } from "./problem.ts";

export type GetAccount409 = Problem & {
  code?: GetAccount409Code;
};
