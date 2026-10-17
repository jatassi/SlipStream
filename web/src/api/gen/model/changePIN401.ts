import type { ChangePIN401Code } from "./changePIN401Code.ts";
import type { Problem } from "./problem.ts";

export type ChangePIN401 = Problem & {
  code?: ChangePIN401Code;
};
