import type { Problem } from "./problem.ts";
import type { ResendSignupLink409Code } from "./resendSignupLink409Code.ts";

export type ResendSignupLink409 = Problem & {
  code?: ResendSignupLink409Code;
};
