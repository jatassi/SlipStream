import type { EndSession500Code } from "./endSession500Code.ts";
import type { Problem } from "./problem.ts";

export type EndSession500 = Problem & {
  code?: EndSession500Code;
};
