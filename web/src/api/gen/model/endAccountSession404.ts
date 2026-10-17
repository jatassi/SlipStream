import type { EndAccountSession404Code } from "./endAccountSession404Code.ts";
import type { Problem } from "./problem.ts";

export type EndAccountSession404 = Problem & {
  code?: EndAccountSession404Code;
};
