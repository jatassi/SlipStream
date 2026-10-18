import type { DeleteProfile500Code } from "./deleteProfile500Code.ts";
import type { Problem } from "./problem.ts";

export type DeleteProfile500 = Problem & {
  code?: DeleteProfile500Code;
};
