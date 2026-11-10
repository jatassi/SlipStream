import type { GetManualSearch404Code } from "./getManualSearch404Code.ts";
import type { Problem } from "./problem.ts";

export type GetManualSearch404 = Problem & {
  code?: GetManualSearch404Code;
};
