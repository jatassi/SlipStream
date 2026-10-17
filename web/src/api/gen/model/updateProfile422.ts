import type { Problem } from "./problem.ts";
import type { UpdateProfile422Code } from "./updateProfile422Code.ts";

export type UpdateProfile422 = Problem & {
  code?: UpdateProfile422Code;
};
