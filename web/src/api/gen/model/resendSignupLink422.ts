import type { Problem } from "./problem.ts";
import type { ResendSignupLink422Code } from "./resendSignupLink422Code.ts";

export type ResendSignupLink422 = Problem & {
  code?: ResendSignupLink422Code;
};
