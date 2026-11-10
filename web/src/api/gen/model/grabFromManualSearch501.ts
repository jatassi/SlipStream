import type { GrabFromManualSearch501Code } from "./grabFromManualSearch501Code.ts";
import type { Problem } from "./problem.ts";

export type GrabFromManualSearch501 = Problem & {
  code?: GrabFromManualSearch501Code;
};
