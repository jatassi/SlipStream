import type { Problem } from "./problem.ts";
import type { TestDownloadClient409Code } from "./testDownloadClient409Code.ts";

export type TestDownloadClient409 = Problem & {
  code?: TestDownloadClient409Code;
};
