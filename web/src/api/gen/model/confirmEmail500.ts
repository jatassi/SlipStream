import type { ConfirmEmail500Code } from "./confirmEmail500Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmEmail500 = Problem & {
  code?: ConfirmEmail500Code;
};
