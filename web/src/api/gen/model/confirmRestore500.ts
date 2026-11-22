import type { ConfirmRestore500Code } from "./confirmRestore500Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmRestore500 = Problem & {
  code?: ConfirmRestore500Code;
};
