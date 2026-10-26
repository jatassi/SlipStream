import type { GetTransferSettings500Code } from "./getTransferSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetTransferSettings500 = Problem & {
  code?: GetTransferSettings500Code;
};
