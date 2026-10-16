import type { ListSignupLinks401Code } from "./listSignupLinks401Code.ts";
import type { Problem } from "./problem.ts";

export type ListSignupLinks401 = Problem & {
  code?: ListSignupLinks401Code;
};
