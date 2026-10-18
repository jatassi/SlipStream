import type { Problem } from "./problem.ts";
import type { SetWorkWants500Code } from "./setWorkWants500Code.ts";

export type SetWorkWants500 = Problem & {
  code?: SetWorkWants500Code;
};
