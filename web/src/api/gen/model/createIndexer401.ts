import type { CreateIndexer401Code } from "./createIndexer401Code.ts";
import type { Problem } from "./problem.ts";

export type CreateIndexer401 = Problem & {
  code?: CreateIndexer401Code;
};
