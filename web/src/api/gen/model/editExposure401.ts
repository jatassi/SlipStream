import type { EditExposure401Code } from "./editExposure401Code.ts";
import type { Problem } from "./problem.ts";

export type EditExposure401 = Problem & {
  code?: EditExposure401Code;
};
