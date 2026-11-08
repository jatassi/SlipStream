import type { Problem } from "./problem.ts";
import type { TestDownloadClient404Code } from "./testDownloadClient404Code.ts";

export type TestDownloadClient404 = Problem & {
  code?: TestDownloadClient404Code;
};
