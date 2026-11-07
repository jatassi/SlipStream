import type { ListTransfers422Code } from "./listTransfers422Code.ts";
import type { Problem } from "./problem.ts";

export type ListTransfers422 = Problem & {
  code?: ListTransfers422Code;
};
