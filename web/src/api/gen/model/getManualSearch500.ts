import type { GetManualSearch500Code } from "./getManualSearch500Code.ts";
import type { Problem } from "./problem.ts";

export type GetManualSearch500 = Problem & {
  code?: GetManualSearch500Code;
};
