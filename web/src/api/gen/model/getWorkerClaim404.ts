import type { GetWorkerClaim404Code } from "./getWorkerClaim404Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkerClaim404 = Problem & {
  code?: GetWorkerClaim404Code;
};
