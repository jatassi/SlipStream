import type { MatchPendingImport404Code } from "./matchPendingImport404Code.ts";
import type { Problem } from "./problem.ts";

export type MatchPendingImport404 = Problem & {
  code?: MatchPendingImport404Code;
};
