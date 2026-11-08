import type { Problem } from "./problem.ts";
import type { TestDownloadClientDraft401Code } from "./testDownloadClientDraft401Code.ts";

export type TestDownloadClientDraft401 = Problem & {
  code?: TestDownloadClientDraft401Code;
};
