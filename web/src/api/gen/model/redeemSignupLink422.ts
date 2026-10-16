import type { Problem } from "./problem.ts";
import type { RedeemSignupLink422Code } from "./redeemSignupLink422Code.ts";

export type RedeemSignupLink422 = Problem & {
  code?: RedeemSignupLink422Code;
};
