import type { EndSession403Code } from "./endSession403Code.ts";
import type { Problem } from "./problem.ts";

export type EndSession403 = Problem & {
  code?: EndSession403Code;
};
