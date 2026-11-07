import type { Problem } from "./problem.ts";
import type { TestIndexer404Code } from "./testIndexer404Code.ts";

export type TestIndexer404 = Problem & {
  code?: TestIndexer404Code;
};
