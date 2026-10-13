import type { CreateProfile404Code } from "./createProfile404Code.ts";
import type { Problem } from "./problem.ts";

export type CreateProfile404 = Problem & {
  code?: CreateProfile404Code;
};
