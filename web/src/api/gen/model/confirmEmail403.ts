import type { ConfirmEmail403Code } from "./confirmEmail403Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmEmail403 = Problem & {
  code?: ConfirmEmail403Code;
};
