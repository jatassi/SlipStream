import type { Problem } from "./problem.ts";
import type { StartPasskeyReconfirm500Code } from "./startPasskeyReconfirm500Code.ts";

export type StartPasskeyReconfirm500 = Problem & {
  code?: StartPasskeyReconfirm500Code;
};
