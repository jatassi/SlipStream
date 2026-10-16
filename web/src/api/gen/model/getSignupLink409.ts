import type { GetSignupLink409Code } from "./getSignupLink409Code.ts";
import type { Problem } from "./problem.ts";

export type GetSignupLink409 = Problem & {
  code?: GetSignupLink409Code;
};
