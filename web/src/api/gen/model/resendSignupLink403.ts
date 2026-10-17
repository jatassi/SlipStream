import type { Problem } from "./problem.ts";
import type { ResendSignupLink403Code } from "./resendSignupLink403Code.ts";

export type ResendSignupLink403 = Problem & {
  code?: ResendSignupLink403Code;
};
