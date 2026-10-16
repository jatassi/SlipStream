import type { GetAddress403Code } from "./getAddress403Code.ts";
import type { Problem } from "./problem.ts";

export type GetAddress403 = Problem & {
  code?: GetAddress403Code;
};
