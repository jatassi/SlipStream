import type { ConfirmRestore404Code } from "./confirmRestore404Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmRestore404 = Problem & {
  code?: ConfirmRestore404Code;
};
