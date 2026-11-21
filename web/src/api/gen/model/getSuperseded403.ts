import type { GetSuperseded403Code } from "./getSuperseded403Code.ts";
import type { Problem } from "./problem.ts";

export type GetSuperseded403 = Problem & {
  code?: GetSuperseded403Code;
};
