import type { EndSession404Code } from "./endSession404Code.ts";
import type { Problem } from "./problem.ts";

export type EndSession404 = Problem & {
  code?: EndSession404Code;
};
