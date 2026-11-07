import type { Problem } from "./problem.ts";
import type { TestIndexer403Code } from "./testIndexer403Code.ts";

export type TestIndexer403 = Problem & {
  code?: TestIndexer403Code;
};
