import type { ConfirmRestore422Code } from "./confirmRestore422Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmRestore422 = Problem & {
  code?: ConfirmRestore422Code;
};
