import type { GetSignupLink401Code } from "./getSignupLink401Code.ts";
import type { Problem } from "./problem.ts";

export type GetSignupLink401 = Problem & {
  code?: GetSignupLink401Code;
};
