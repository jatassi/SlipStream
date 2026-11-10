import type { GrabFromManualSearch422Code } from "./grabFromManualSearch422Code.ts";
import type { Problem } from "./problem.ts";

export type GrabFromManualSearch422 = Problem & {
  code?: GrabFromManualSearch422Code;
};
