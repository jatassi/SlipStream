import type { Problem } from "./problem.ts";
import type { ReconfirmSession409Code } from "./reconfirmSession409Code.ts";

export type ReconfirmSession409 = Problem & {
  code?: ReconfirmSession409Code;
};
