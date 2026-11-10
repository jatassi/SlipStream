import type { GetManualSearch403Code } from "./getManualSearch403Code.ts";
import type { Problem } from "./problem.ts";

export type GetManualSearch403 = Problem & {
  code?: GetManualSearch403Code;
};
