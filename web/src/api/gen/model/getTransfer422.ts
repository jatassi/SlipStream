import type { GetTransfer422Code } from "./getTransfer422Code.ts";
import type { Problem } from "./problem.ts";

export type GetTransfer422 = Problem & {
  code?: GetTransfer422Code;
};
