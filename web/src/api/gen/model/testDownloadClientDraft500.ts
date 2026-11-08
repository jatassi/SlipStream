import type { Problem } from "./problem.ts";
import type { TestDownloadClientDraft500Code } from "./testDownloadClientDraft500Code.ts";

export type TestDownloadClientDraft500 = Problem & {
  code?: TestDownloadClientDraft500Code;
};
