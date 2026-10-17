import type { DeletePasskey403Code } from "./deletePasskey403Code.ts";
import type { Problem } from "./problem.ts";

export type DeletePasskey403 = Problem & {
  code?: DeletePasskey403Code;
};
