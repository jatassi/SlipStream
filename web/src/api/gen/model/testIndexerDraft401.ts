import type { Problem } from "./problem.ts";
import type { TestIndexerDraft401Code } from "./testIndexerDraft401Code.ts";

export type TestIndexerDraft401 = Problem & {
  code?: TestIndexerDraft401Code;
};
