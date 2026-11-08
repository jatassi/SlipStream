import type { Problem } from "./problem.ts";
import type { TestDownloadClientDraft502Code } from "./testDownloadClientDraft502Code.ts";

export type TestDownloadClientDraft502 = Problem & {
  code?: TestDownloadClientDraft502Code;
};
