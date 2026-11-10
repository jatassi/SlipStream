import type { GetManualSearch401Code } from "./getManualSearch401Code.ts";
import type { Problem } from "./problem.ts";

export type GetManualSearch401 = Problem & {
  code?: GetManualSearch401Code;
};
