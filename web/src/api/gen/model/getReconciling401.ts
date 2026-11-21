import type { GetReconciling401Code } from "./getReconciling401Code.ts";
import type { Problem } from "./problem.ts";

export type GetReconciling401 = Problem & {
  code?: GetReconciling401Code;
};
