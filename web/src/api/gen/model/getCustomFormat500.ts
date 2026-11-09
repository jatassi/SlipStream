import type { GetCustomFormat500Code } from "./getCustomFormat500Code.ts";
import type { Problem } from "./problem.ts";

export type GetCustomFormat500 = Problem & {
  code?: GetCustomFormat500Code;
};
