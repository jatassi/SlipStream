import type { Problem } from "./problem.ts";
import type { UpdateProfile409Code } from "./updateProfile409Code.ts";

export type UpdateProfile409 = Problem & {
  code?: UpdateProfile409Code;
};
