import type { AddExposure404Code } from "./addExposure404Code.ts";
import type { Problem } from "./problem.ts";

export type AddExposure404 = Problem & {
  code?: AddExposure404Code;
};
