import type { CreateSignupLink501Code } from "./createSignupLink501Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSignupLink501 = Problem & {
  code?: CreateSignupLink501Code;
};
