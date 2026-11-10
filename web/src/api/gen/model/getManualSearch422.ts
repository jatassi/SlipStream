import type { GetManualSearch422Code } from "./getManualSearch422Code.ts";
import type { Problem } from "./problem.ts";

export type GetManualSearch422 = Problem & {
  code?: GetManualSearch422Code;
};
