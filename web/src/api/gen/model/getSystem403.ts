import type { GetSystem403Code } from "./getSystem403Code.ts";
import type { Problem } from "./problem.ts";

export type GetSystem403 = Problem & {
  code?: GetSystem403Code;
};
