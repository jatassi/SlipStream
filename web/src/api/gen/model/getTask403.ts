import type { GetTask403Code } from "./getTask403Code.ts";
import type { Problem } from "./problem.ts";

export type GetTask403 = Problem & {
  code?: GetTask403Code;
};
