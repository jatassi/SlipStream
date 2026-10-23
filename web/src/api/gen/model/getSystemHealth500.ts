import type { GetSystemHealth500Code } from "./getSystemHealth500Code.ts";
import type { Problem } from "./problem.ts";

export type GetSystemHealth500 = Problem & {
  code?: GetSystemHealth500Code;
};
