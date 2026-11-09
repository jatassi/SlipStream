import type { MatchPendingImport401Code } from "./matchPendingImport401Code.ts";
import type { Problem } from "./problem.ts";

export type MatchPendingImport401 = Problem & {
  code?: MatchPendingImport401Code;
};
