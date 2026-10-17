import type { CreateSignupLink500Code } from "./createSignupLink500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSignupLink500 = Problem & {
  code?: CreateSignupLink500Code;
};
