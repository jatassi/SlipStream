import type { EndAccountSession403Code } from "./endAccountSession403Code.ts";
import type { Problem } from "./problem.ts";

export type EndAccountSession403 = Problem & {
  code?: EndAccountSession403Code;
};
