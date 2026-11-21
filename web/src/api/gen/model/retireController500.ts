import type { Problem } from "./problem.ts";
import type { RetireController500Code } from "./retireController500Code.ts";

export type RetireController500 = Problem & {
  code?: RetireController500Code;
};
