import type { ChangePIN500Code } from "./changePIN500Code.ts";
import type { Problem } from "./problem.ts";

export type ChangePIN500 = Problem & {
  code?: ChangePIN500Code;
};
