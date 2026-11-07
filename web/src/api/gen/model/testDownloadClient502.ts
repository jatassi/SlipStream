import type { Problem } from "./problem.ts";
import type { TestDownloadClient502Code } from "./testDownloadClient502Code.ts";

export type TestDownloadClient502 = Problem & {
  code?: TestDownloadClient502Code;
};
