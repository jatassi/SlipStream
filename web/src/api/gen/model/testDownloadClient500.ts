import type { Problem } from "./problem.ts";
import type { TestDownloadClient500Code } from "./testDownloadClient500Code.ts";

export type TestDownloadClient500 = Problem & {
  code?: TestDownloadClient500Code;
};
