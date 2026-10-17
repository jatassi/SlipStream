import type { Problem } from "./problem.ts";
import type { ReconfirmSession401Code } from "./reconfirmSession401Code.ts";

export type ReconfirmSession401 = Problem & {
  code?: ReconfirmSession401Code;
};
