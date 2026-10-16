import type { CreateSignupLink422Code } from "./createSignupLink422Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSignupLink422 = Problem & {
  code?: CreateSignupLink422Code;
};
