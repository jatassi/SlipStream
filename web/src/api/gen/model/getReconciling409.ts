import type { GetReconciling409Code } from "./getReconciling409Code.ts";
import type { Problem } from "./problem.ts";

export type GetReconciling409 = Problem & {
  code?: GetReconciling409Code;
};
