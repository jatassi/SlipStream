import type { Problem } from "./problem.ts";
import type { StartPasskeyReconfirm409Code } from "./startPasskeyReconfirm409Code.ts";

export type StartPasskeyReconfirm409 = Problem & {
  code?: StartPasskeyReconfirm409Code;
};
