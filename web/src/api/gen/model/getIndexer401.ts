import type { GetIndexer401Code } from "./getIndexer401Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexer401 = Problem & {
  code?: GetIndexer401Code;
};
