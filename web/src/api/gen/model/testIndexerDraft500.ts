import type { Problem } from "./problem.ts";
import type { TestIndexerDraft500Code } from "./testIndexerDraft500Code.ts";

export type TestIndexerDraft500 = Problem & {
  code?: TestIndexerDraft500Code;
};
