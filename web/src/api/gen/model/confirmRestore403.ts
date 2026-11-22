import type { ConfirmRestore403Code } from "./confirmRestore403Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmRestore403 = Problem & {
  code?: ConfirmRestore403Code;
};
