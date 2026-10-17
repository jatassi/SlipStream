import type { Problem } from "./problem.ts";
import type { ResendSignupLink404Code } from "./resendSignupLink404Code.ts";

export type ResendSignupLink404 = Problem & {
  code?: ResendSignupLink404Code;
};
