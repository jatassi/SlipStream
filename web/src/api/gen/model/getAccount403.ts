import type { GetAccount403Code } from "./getAccount403Code.ts";
import type { Problem } from "./problem.ts";

export type GetAccount403 = Problem & {
  code?: GetAccount403Code;
};
