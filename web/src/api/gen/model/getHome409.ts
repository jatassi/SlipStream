import type { GetHome409Code } from "./getHome409Code.ts";
import type { Problem } from "./problem.ts";

export type GetHome409 = Problem & {
  code?: GetHome409Code;
};
