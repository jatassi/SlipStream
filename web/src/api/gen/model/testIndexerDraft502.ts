import type { Problem } from "./problem.ts";
import type { TestIndexerDraft502Code } from "./testIndexerDraft502Code.ts";

export type TestIndexerDraft502 = Problem & {
  code?: TestIndexerDraft502Code;
};
