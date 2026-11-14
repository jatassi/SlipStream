import type { AddMediaServer500Code } from "./addMediaServer500Code.ts";
import type { Problem } from "./problem.ts";

export type AddMediaServer500 = Problem & {
  code?: AddMediaServer500Code;
};
