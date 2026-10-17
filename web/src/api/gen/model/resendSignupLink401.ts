import type { Problem } from "./problem.ts";
import type { ResendSignupLink401Code } from "./resendSignupLink401Code.ts";

export type ResendSignupLink401 = Problem & {
  code?: ResendSignupLink401Code;
};
