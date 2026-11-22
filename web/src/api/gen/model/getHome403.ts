import type { GetHome403Code } from "./getHome403Code.ts";
import type { Problem } from "./problem.ts";

export type GetHome403 = Problem & {
  code?: GetHome403Code;
};
