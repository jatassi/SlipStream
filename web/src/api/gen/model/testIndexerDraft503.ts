import type { Problem } from "./problem.ts";
import type { TestIndexerDraft503Code } from "./testIndexerDraft503Code.ts";

export type TestIndexerDraft503 = Problem & {
  code?: TestIndexerDraft503Code;
};
