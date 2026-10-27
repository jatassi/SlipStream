import type { CreateIndexer404Code } from "./createIndexer404Code.ts";
import type { Problem } from "./problem.ts";

export type CreateIndexer404 = Problem & {
  code?: CreateIndexer404Code;
};
