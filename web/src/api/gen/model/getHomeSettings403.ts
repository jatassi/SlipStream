import type { GetHomeSettings403Code } from "./getHomeSettings403Code.ts";
import type { Problem } from "./problem.ts";

export type GetHomeSettings403 = Problem & {
  code?: GetHomeSettings403Code;
};
