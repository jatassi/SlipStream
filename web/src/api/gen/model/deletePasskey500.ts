import type { DeletePasskey500Code } from "./deletePasskey500Code.ts";
import type { Problem } from "./problem.ts";

export type DeletePasskey500 = Problem & {
  code?: DeletePasskey500Code;
};
