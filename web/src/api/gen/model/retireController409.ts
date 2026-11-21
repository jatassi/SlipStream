import type { Problem } from "./problem.ts";
import type { RetireController409Code } from "./retireController409Code.ts";

export type RetireController409 = Problem & {
  code?: RetireController409Code;
};
