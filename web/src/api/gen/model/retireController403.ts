import type { Problem } from "./problem.ts";
import type { RetireController403Code } from "./retireController403Code.ts";

export type RetireController403 = Problem & {
  code?: RetireController403Code;
};
