import type { GetManualSearch501Code } from "./getManualSearch501Code.ts";
import type { Problem } from "./problem.ts";

export type GetManualSearch501 = Problem & {
  code?: GetManualSearch501Code;
};
