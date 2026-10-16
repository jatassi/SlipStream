import type { Problem } from "./problem.ts";
import type { StartPasskeyReconfirm403Code } from "./startPasskeyReconfirm403Code.ts";

export type StartPasskeyReconfirm403 = Problem & {
  code?: StartPasskeyReconfirm403Code;
};
