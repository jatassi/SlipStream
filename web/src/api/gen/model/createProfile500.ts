import type { CreateProfile500Code } from "./createProfile500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateProfile500 = Problem & {
  code?: CreateProfile500Code;
};
