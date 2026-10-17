import type { ChangePIN409Code } from "./changePIN409Code.ts";
import type { Problem } from "./problem.ts";

export type ChangePIN409 = Problem & {
  code?: ChangePIN409Code;
};
