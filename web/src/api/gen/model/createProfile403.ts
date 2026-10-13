import type { CreateProfile403Code } from "./createProfile403Code.ts";
import type { Problem } from "./problem.ts";

export type CreateProfile403 = Problem & {
  code?: CreateProfile403Code;
};
