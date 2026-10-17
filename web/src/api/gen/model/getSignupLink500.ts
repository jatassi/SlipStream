import type { GetSignupLink500Code } from "./getSignupLink500Code.ts";
import type { Problem } from "./problem.ts";

export type GetSignupLink500 = Problem & {
  code?: GetSignupLink500Code;
};
