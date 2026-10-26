import type { GetTransfer404Code } from "./getTransfer404Code.ts";
import type { Problem } from "./problem.ts";

export type GetTransfer404 = Problem & {
  code?: GetTransfer404Code;
};
