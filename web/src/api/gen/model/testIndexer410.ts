import type { Problem } from "./problem.ts";
import type { TestIndexer410Code } from "./testIndexer410Code.ts";

export type TestIndexer410 = Problem & {
  code?: TestIndexer410Code;
};
