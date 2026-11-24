import type { EditExposure403Code } from "./editExposure403Code.ts";
import type { Problem } from "./problem.ts";

export type EditExposure403 = Problem & {
  code?: EditExposure403Code;
};
