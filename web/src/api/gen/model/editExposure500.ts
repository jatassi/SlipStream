import type { EditExposure500Code } from "./editExposure500Code.ts";
import type { Problem } from "./problem.ts";

export type EditExposure500 = Problem & {
  code?: EditExposure500Code;
};
