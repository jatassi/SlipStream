import type { Problem } from "./problem.ts";
import type { TestDownloadClient401Code } from "./testDownloadClient401Code.ts";

export type TestDownloadClient401 = Problem & {
  code?: TestDownloadClient401Code;
};
