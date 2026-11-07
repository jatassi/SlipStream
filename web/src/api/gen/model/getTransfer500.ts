import type { GetTransfer500Code } from "./getTransfer500Code.ts";
import type { Problem } from "./problem.ts";

export type GetTransfer500 = Problem & {
  code?: GetTransfer500Code;
};
