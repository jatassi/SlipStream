import type { Problem } from "./problem.ts";
import type { TestIndexer401Code } from "./testIndexer401Code.ts";

export type TestIndexer401 = Problem & {
  code?: TestIndexer401Code;
};
