import type { Problem } from "./problem.ts";
import type { TestIndexer409Code } from "./testIndexer409Code.ts";

export type TestIndexer409 = Problem & {
  code?: TestIndexer409Code;
};
