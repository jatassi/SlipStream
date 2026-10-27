import type { Problem } from "./problem.ts";
import type { TestIndexer422Code } from "./testIndexer422Code.ts";

export type TestIndexer422 = Problem & {
  code?: TestIndexer422Code;
};
