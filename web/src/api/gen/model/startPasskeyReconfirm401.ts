import type { Problem } from "./problem.ts";
import type { StartPasskeyReconfirm401Code } from "./startPasskeyReconfirm401Code.ts";

export type StartPasskeyReconfirm401 = Problem & {
  code?: StartPasskeyReconfirm401Code;
};
