import type { CreateSession429Code } from "./createSession429Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSession429 = Problem & {
  code?: CreateSession429Code;
};
