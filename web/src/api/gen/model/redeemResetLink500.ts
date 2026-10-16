import type { Problem } from "./problem.ts";
import type { RedeemResetLink500Code } from "./redeemResetLink500Code.ts";

export type RedeemResetLink500 = Problem & {
  code?: RedeemResetLink500Code;
};
