import type { Problem } from "./problem.ts";
import type { UpdateProfile401Code } from "./updateProfile401Code.ts";

export type UpdateProfile401 = Problem & {
  code?: UpdateProfile401Code;
};
