import type { EndAccountSession422Code } from "./endAccountSession422Code.ts";
import type { Problem } from "./problem.ts";

export type EndAccountSession422 = Problem & {
  code?: EndAccountSession422Code;
};
