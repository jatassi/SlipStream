import type { GetIndexer500Code } from "./getIndexer500Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexer500 = Problem & {
  code?: GetIndexer500Code;
};
