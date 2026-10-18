import type { GetWorkWants500Code } from "./getWorkWants500Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkWants500 = Problem & {
  code?: GetWorkWants500Code;
};
