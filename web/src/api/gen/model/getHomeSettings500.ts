import type { GetHomeSettings500Code } from "./getHomeSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetHomeSettings500 = Problem & {
  code?: GetHomeSettings500Code;
};
