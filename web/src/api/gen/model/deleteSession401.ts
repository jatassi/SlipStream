import type { DeleteSession401Code } from "./deleteSession401Code.ts";
import type { Problem } from "./problem.ts";

export type DeleteSession401 = Problem & {
  code?: DeleteSession401Code;
};
