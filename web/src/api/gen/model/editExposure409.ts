import type { EditExposure409Code } from "./editExposure409Code.ts";
import type { Problem } from "./problem.ts";

export type EditExposure409 = Problem & {
  code?: EditExposure409Code;
};
