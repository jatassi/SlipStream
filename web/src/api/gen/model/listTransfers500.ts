import type { ListTransfers500Code } from "./listTransfers500Code.ts";
import type { Problem } from "./problem.ts";

export type ListTransfers500 = Problem & {
  code?: ListTransfers500Code;
};
