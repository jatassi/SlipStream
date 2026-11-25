import type { Problem } from "./problem.ts";
import type { RemoveExposure404Code } from "./removeExposure404Code.ts";

export type RemoveExposure404 = Problem & {
  code?: RemoveExposure404Code;
};
