import type { GetTransfer403Code } from "./getTransfer403Code.ts";
import type { Problem } from "./problem.ts";

export type GetTransfer403 = Problem & {
  code?: GetTransfer403Code;
};
