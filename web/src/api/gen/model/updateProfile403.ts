import type { Problem } from "./problem.ts";
import type { UpdateProfile403Code } from "./updateProfile403Code.ts";

export type UpdateProfile403 = Problem & {
  code?: UpdateProfile403Code;
};
