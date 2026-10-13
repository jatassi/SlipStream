import type { CreateProfile401Code } from "./createProfile401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateProfile401 = Problem & {
  code?: CreateProfile401Code;
};
