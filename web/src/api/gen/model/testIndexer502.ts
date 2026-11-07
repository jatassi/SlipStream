import type { Problem } from "./problem.ts";
import type { TestIndexer502Code } from "./testIndexer502Code.ts";

export type TestIndexer502 = Problem & {
  code?: TestIndexer502Code;
};
