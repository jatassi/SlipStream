import type { GetSignupLink403Code } from "./getSignupLink403Code.ts";
import type { Problem } from "./problem.ts";

export type GetSignupLink403 = Problem & {
  code?: GetSignupLink403Code;
};
