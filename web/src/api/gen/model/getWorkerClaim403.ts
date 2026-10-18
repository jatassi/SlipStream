import type { GetWorkerClaim403Code } from "./getWorkerClaim403Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkerClaim403 = Problem & {
  code?: GetWorkerClaim403Code;
};
