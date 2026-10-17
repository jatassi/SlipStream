import type { Problem } from "./problem.ts";
import type { RedeemSignupLink409Code } from "./redeemSignupLink409Code.ts";

export type RedeemSignupLink409 = Problem & {
  code?: RedeemSignupLink409Code;
};
