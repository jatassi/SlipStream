import type { Problem } from "./problem.ts";
import type { RedeemResetLink403Code } from "./redeemResetLink403Code.ts";

export type RedeemResetLink403 = Problem & {
  code?: RedeemResetLink403Code;
};
