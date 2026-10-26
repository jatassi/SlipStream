import type { ListTransfers401Code } from "./listTransfers401Code.ts";
import type { Problem } from "./problem.ts";

export type ListTransfers401 = Problem & {
  code?: ListTransfers401Code;
};
