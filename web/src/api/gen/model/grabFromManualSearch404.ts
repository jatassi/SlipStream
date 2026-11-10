import type { GrabFromManualSearch404Code } from "./grabFromManualSearch404Code.ts";
import type { Problem } from "./problem.ts";

export type GrabFromManualSearch404 = Problem & {
  code?: GrabFromManualSearch404Code;
};
