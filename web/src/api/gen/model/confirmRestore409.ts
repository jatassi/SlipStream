import type { ConfirmRestore409Code } from "./confirmRestore409Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmRestore409 = Problem & {
  code?: ConfirmRestore409Code;
};
