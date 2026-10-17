import type { Problem } from "./problem.ts";
import type { RedeemResetLink422Code } from "./redeemResetLink422Code.ts";

export type RedeemResetLink422 = Problem & {
  code?: RedeemResetLink422Code;
};
