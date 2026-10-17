import type { ChangePIN422Code } from "./changePIN422Code.ts";
import type { Problem } from "./problem.ts";

export type ChangePIN422 = Problem & {
  code?: ChangePIN422Code;
};
