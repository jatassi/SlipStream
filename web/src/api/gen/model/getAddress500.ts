import type { GetAddress500Code } from "./getAddress500Code.ts";
import type { Problem } from "./problem.ts";

export type GetAddress500 = Problem & {
  code?: GetAddress500Code;
};
