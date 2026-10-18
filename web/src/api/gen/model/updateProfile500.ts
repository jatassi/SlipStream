import type { Problem } from "./problem.ts";
import type { UpdateProfile500Code } from "./updateProfile500Code.ts";

export type UpdateProfile500 = Problem & {
  code?: UpdateProfile500Code;
};
