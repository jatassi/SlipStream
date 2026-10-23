import type { GetSystemHealth403Code } from "./getSystemHealth403Code.ts";
import type { Problem } from "./problem.ts";

export type GetSystemHealth403 = Problem & {
  code?: GetSystemHealth403Code;
};
