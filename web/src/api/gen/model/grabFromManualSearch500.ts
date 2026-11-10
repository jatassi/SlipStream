import type { GrabFromManualSearch500Code } from "./grabFromManualSearch500Code.ts";
import type { Problem } from "./problem.ts";

export type GrabFromManualSearch500 = Problem & {
  code?: GrabFromManualSearch500Code;
};
