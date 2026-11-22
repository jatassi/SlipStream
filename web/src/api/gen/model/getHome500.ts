import type { GetHome500Code } from "./getHome500Code.ts";
import type { Problem } from "./problem.ts";

export type GetHome500 = Problem & {
  code?: GetHome500Code;
};
