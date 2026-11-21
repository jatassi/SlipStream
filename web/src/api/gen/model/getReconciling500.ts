import type { GetReconciling500Code } from "./getReconciling500Code.ts";
import type { Problem } from "./problem.ts";

export type GetReconciling500 = Problem & {
  code?: GetReconciling500Code;
};
