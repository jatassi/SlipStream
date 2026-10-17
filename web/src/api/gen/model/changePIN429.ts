import type { ChangePIN429Code } from "./changePIN429Code.ts";
import type { Problem } from "./problem.ts";

export type ChangePIN429 = Problem & {
  code?: ChangePIN429Code;
};
