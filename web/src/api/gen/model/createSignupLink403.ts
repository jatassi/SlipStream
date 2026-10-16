import type { CreateSignupLink403Code } from "./createSignupLink403Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSignupLink403 = Problem & {
  code?: CreateSignupLink403Code;
};
