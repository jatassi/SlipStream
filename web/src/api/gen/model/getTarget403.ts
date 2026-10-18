import type { GetTarget403Code } from "./getTarget403Code.ts";
import type { Problem } from "./problem.ts";

export type GetTarget403 = Problem & {
  code?: GetTarget403Code;
};
