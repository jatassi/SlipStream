import type { ChangePIN403Code } from "./changePIN403Code.ts";
import type { Problem } from "./problem.ts";

export type ChangePIN403 = Problem & {
  code?: ChangePIN403Code;
};
