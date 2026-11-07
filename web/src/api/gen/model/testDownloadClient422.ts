import type { Problem } from "./problem.ts";
import type { TestDownloadClient422Code } from "./testDownloadClient422Code.ts";

export type TestDownloadClient422 = Problem & {
  code?: TestDownloadClient422Code;
  /** @nullable */
  fields?: string[] | null;
};
