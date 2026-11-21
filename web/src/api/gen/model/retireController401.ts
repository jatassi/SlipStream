import type { Problem } from "./problem.ts";
import type { RetireController401Code } from "./retireController401Code.ts";

export type RetireController401 = Problem & {
  code?: RetireController401Code;
};
