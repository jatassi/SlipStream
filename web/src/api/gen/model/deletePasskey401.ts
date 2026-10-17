import type { DeletePasskey401Code } from "./deletePasskey401Code.ts";
import type { Problem } from "./problem.ts";

export type DeletePasskey401 = Problem & {
  code?: DeletePasskey401Code;
};
