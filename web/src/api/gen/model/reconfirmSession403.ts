import type { Problem } from "./problem.ts";
import type { ReconfirmSession403Code } from "./reconfirmSession403Code.ts";

export type ReconfirmSession403 = Problem & {
  code?: ReconfirmSession403Code;
};
