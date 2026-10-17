import type { EndAccountSession401Code } from "./endAccountSession401Code.ts";
import type { Problem } from "./problem.ts";

export type EndAccountSession401 = Problem & {
  code?: EndAccountSession401Code;
};
