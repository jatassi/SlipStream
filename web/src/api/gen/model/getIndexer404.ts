import type { GetIndexer404Code } from "./getIndexer404Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexer404 = Problem & {
  code?: GetIndexer404Code;
};
