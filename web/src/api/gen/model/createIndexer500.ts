import type { CreateIndexer500Code } from "./createIndexer500Code.ts";
import type { Problem } from "./problem.ts";

export type CreateIndexer500 = Problem & {
  code?: CreateIndexer500Code;
};
