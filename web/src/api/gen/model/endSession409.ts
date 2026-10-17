import type { EndSession409Code } from "./endSession409Code.ts";
import type { Problem } from "./problem.ts";

export type EndSession409 = Problem & {
  code?: EndSession409Code;
};
