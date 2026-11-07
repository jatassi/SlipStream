import type { GetTransfer409Code } from "./getTransfer409Code.ts";
import type { Problem } from "./problem.ts";

export type GetTransfer409 = Problem & {
  code?: GetTransfer409Code;
};
