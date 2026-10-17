import type { Problem } from "./problem.ts";
import type { StartPasskeyReconfirm429Code } from "./startPasskeyReconfirm429Code.ts";

export type StartPasskeyReconfirm429 = Problem & {
  code?: StartPasskeyReconfirm429Code;
};
