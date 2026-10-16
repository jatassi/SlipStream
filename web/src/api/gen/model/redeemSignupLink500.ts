import type { Problem } from "./problem.ts";
import type { RedeemSignupLink500Code } from "./redeemSignupLink500Code.ts";

export type RedeemSignupLink500 = Problem & {
  code?: RedeemSignupLink500Code;
};
