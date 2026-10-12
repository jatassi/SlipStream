import type { CreateSession401Code } from "./createSession401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSession401 = Problem & {
  code?: CreateSession401Code;
};
