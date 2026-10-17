import type { Problem } from "./problem.ts";
import type { UpdatePasskey403Code } from "./updatePasskey403Code.ts";

export type UpdatePasskey403 = Problem & {
  code?: UpdatePasskey403Code;
};
