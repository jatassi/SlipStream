import type { GetIndexerSettings500Code } from "./getIndexerSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerSettings500 = Problem & {
  code?: GetIndexerSettings500Code;
};
