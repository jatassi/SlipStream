import type { GetNode403Code } from "./getNode403Code.ts";
import type { Problem } from "./problem.ts";

export type GetNode403 = Problem & {
  code?: GetNode403Code;
};
