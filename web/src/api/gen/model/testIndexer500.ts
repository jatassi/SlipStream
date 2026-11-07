import type { Problem } from "./problem.ts";
import type { TestIndexer500Code } from "./testIndexer500Code.ts";

export type TestIndexer500 = Problem & {
  code?: TestIndexer500Code;
};
