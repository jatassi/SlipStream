import type { CreateSignupLink401Code } from "./createSignupLink401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSignupLink401 = Problem & {
  code?: CreateSignupLink401Code;
};
