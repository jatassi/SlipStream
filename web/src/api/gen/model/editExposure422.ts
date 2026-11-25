import type { EditExposure422Code } from "./editExposure422Code.ts";
import type { Problem } from "./problem.ts";

export type EditExposure422 = Problem & {
  code?: EditExposure422Code;
};
