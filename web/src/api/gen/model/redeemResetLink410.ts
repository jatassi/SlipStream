import type { Problem } from "./problem.ts";
import type { RedeemResetLink410Code } from "./redeemResetLink410Code.ts";

export type RedeemResetLink410 = Problem & {
  code?: RedeemResetLink410Code;
};
