import type { Problem } from "./problem.ts";
import type { TestDownloadClient403Code } from "./testDownloadClient403Code.ts";

export type TestDownloadClient403 = Problem & {
  code?: TestDownloadClient403Code;
};
