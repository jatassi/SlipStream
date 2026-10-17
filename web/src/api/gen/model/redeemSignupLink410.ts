import type { Problem } from "./problem.ts";
import type { RedeemSignupLink410Code } from "./redeemSignupLink410Code.ts";

export type RedeemSignupLink410 = Problem & {
  code?: RedeemSignupLink410Code;
};
