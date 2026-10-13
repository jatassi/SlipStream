import type { CreateProfile422Code } from "./createProfile422Code.ts";
import type { Problem } from "./problem.ts";

export type CreateProfile422 = Problem & {
  code?: CreateProfile422Code;
};
