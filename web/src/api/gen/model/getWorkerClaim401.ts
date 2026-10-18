import type { GetWorkerClaim401Code } from "./getWorkerClaim401Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkerClaim401 = Problem & {
  code?: GetWorkerClaim401Code;
};
