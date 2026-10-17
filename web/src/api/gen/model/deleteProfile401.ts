import type { DeleteProfile401Code } from "./deleteProfile401Code.ts";
import type { Problem } from "./problem.ts";

export type DeleteProfile401 = Problem & {
  code?: DeleteProfile401Code;
};
