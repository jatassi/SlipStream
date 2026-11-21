import type { ConfirmRestore401Code } from "./confirmRestore401Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmRestore401 = Problem & {
  code?: ConfirmRestore401Code;
};
