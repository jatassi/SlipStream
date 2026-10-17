import type { Problem } from "./problem.ts";
import type { ReconfirmSession429Code } from "./reconfirmSession429Code.ts";

export type ReconfirmSession429 = Problem & {
  code?: ReconfirmSession429Code;
};
