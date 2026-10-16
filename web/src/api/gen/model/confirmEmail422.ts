import type { ConfirmEmail422Code } from "./confirmEmail422Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmEmail422 = Problem & {
  code?: ConfirmEmail422Code;
};
