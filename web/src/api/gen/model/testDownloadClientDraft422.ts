import type { Problem } from "./problem.ts";
import type { TestDownloadClientDraft422Code } from "./testDownloadClientDraft422Code.ts";

export type TestDownloadClientDraft422 = Problem & {
  code?: TestDownloadClientDraft422Code;
  /** @nullable */
  fields?: string[] | null;
};
