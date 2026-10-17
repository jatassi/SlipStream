import type { GetSignupLink422Code } from "./getSignupLink422Code.ts";
import type { Problem } from "./problem.ts";

export type GetSignupLink422 = Problem & {
  code?: GetSignupLink422Code;
};
