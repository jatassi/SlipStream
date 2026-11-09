import type { MatchPendingImport500Code } from "./matchPendingImport500Code.ts";
import type { Problem } from "./problem.ts";

export type MatchPendingImport500 = Problem & {
  code?: MatchPendingImport500Code;
};
