import type { Problem } from "./problem.ts";
import type { TestIndexerDraft410Code } from "./testIndexerDraft410Code.ts";

export type TestIndexerDraft410 = Problem & {
  code?: TestIndexerDraft410Code;
};
