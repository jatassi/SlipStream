import type { ListAccounts403Code } from "./listAccounts403Code.ts";
import type { Problem } from "./problem.ts";

export type ListAccounts403 = Problem & {
  code?: ListAccounts403Code;
};
