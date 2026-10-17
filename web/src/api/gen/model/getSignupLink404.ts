import type { GetSignupLink404Code } from "./getSignupLink404Code.ts";
import type { Problem } from "./problem.ts";

export type GetSignupLink404 = Problem & {
  code?: GetSignupLink404Code;
};
