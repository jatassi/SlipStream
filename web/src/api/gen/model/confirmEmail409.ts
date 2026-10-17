import type { ConfirmEmail409Code } from "./confirmEmail409Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmEmail409 = Problem & {
  code?: ConfirmEmail409Code;
};
