import type { GetWorkerClaim500Code } from "./getWorkerClaim500Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkerClaim500 = Problem & {
  code?: GetWorkerClaim500Code;
};
