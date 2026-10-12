import type { CreateSession500Code } from "./createSession500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSession500 = Problem & {
  code?: CreateSession500Code;
};
