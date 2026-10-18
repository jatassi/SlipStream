import type { Problem } from "./problem.ts";
import type { UpdateProfile404Code } from "./updateProfile404Code.ts";

export type UpdateProfile404 = Problem & {
  code?: UpdateProfile404Code;
};
