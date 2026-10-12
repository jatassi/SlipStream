import type { GetStatus500Code } from "./getStatus500Code.ts";
import type { Problem } from "./problem.ts";

export type GetStatus500 = Problem & {
  code?: GetStatus500Code;
};
