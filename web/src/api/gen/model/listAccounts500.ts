import type { ListAccounts500Code } from "./listAccounts500Code.ts";
import type { Problem } from "./problem.ts";

export type ListAccounts500 = Problem & {
  code?: ListAccounts500Code;
};
