import type { Problem } from "./problem.ts";
import type { TestIndexerDraft409Code } from "./testIndexerDraft409Code.ts";

export type TestIndexerDraft409 = Problem & {
  code?: TestIndexerDraft409Code;
};
