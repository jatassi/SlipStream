import type { ListAccounts401Code } from "./listAccounts401Code.ts";
import type { Problem } from "./problem.ts";

export type ListAccounts401 = Problem & {
  code?: ListAccounts401Code;
};
