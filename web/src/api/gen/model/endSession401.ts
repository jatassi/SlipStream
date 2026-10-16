import type { EndSession401Code } from "./endSession401Code.ts";
import type { Problem } from "./problem.ts";

export type EndSession401 = Problem & {
  code?: EndSession401Code;
};
