import type { Problem } from "./problem.ts";
import type { ResumeSupersededController500Code } from "./resumeSupersededController500Code.ts";

export type ResumeSupersededController500 = Problem & {
  code?: ResumeSupersededController500Code;
};
