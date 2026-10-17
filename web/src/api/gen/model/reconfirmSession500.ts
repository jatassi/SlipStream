import type { Problem } from "./problem.ts";
import type { ReconfirmSession500Code } from "./reconfirmSession500Code.ts";

export type ReconfirmSession500 = Problem & {
  code?: ReconfirmSession500Code;
};
