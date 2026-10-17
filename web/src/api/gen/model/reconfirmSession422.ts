import type { Problem } from "./problem.ts";
import type { ReconfirmSession422Code } from "./reconfirmSession422Code.ts";

export type ReconfirmSession422 = Problem & {
  code?: ReconfirmSession422Code;
};
