import type { ConfirmEmail410Code } from "./confirmEmail410Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmEmail410 = Problem & {
  code?: ConfirmEmail410Code;
};
