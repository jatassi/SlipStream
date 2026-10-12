import type { DeleteSession500Code } from "./deleteSession500Code.ts";
import type { Problem } from "./problem.ts";

export type DeleteSession500 = Problem & {
  code?: DeleteSession500Code;
};
