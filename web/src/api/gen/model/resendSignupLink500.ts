import type { Problem } from "./problem.ts";
import type { ResendSignupLink500Code } from "./resendSignupLink500Code.ts";

export type ResendSignupLink500 = Problem & {
  code?: ResendSignupLink500Code;
};
