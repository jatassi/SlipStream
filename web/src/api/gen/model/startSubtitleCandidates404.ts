import type { Problem } from "./problem.ts";
import type { StartSubtitleCandidates404Code } from "./startSubtitleCandidates404Code.ts";

export type StartSubtitleCandidates404 = Problem & {
  code?: StartSubtitleCandidates404Code;
};
