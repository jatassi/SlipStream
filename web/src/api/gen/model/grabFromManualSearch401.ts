import type { GrabFromManualSearch401Code } from "./grabFromManualSearch401Code.ts";
import type { Problem } from "./problem.ts";

export type GrabFromManualSearch401 = Problem & {
  code?: GrabFromManualSearch401Code;
};
