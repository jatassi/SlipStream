import type { Problem } from "./problem.ts";
import type { TestIndexerDraft404Code } from "./testIndexerDraft404Code.ts";

export type TestIndexerDraft404 = Problem & {
  code?: TestIndexerDraft404Code;
};
