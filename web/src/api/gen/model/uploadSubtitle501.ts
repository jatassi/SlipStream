import type { Problem } from "./problem.ts";
import type { UploadSubtitle501Code } from "./uploadSubtitle501Code.ts";

export type UploadSubtitle501 = Problem & {
  code?: UploadSubtitle501Code;
};
