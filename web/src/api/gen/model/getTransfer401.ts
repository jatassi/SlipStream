import type { GetTransfer401Code } from "./getTransfer401Code.ts";
import type { Problem } from "./problem.ts";

export type GetTransfer401 = Problem & {
  code?: GetTransfer401Code;
};
