import type { EndAccountSession500Code } from "./endAccountSession500Code.ts";
import type { Problem } from "./problem.ts";

export type EndAccountSession500 = Problem & {
  code?: EndAccountSession500Code;
};
