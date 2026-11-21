import type { Problem } from "./problem.ts";
import type { UploadSubtitle403Code } from "./uploadSubtitle403Code.ts";

export type UploadSubtitle403 = Problem & {
  code?: UploadSubtitle403Code;
};
