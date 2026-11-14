import type { AddMediaServer401Code } from "./addMediaServer401Code.ts";
import type { Problem } from "./problem.ts";

export type AddMediaServer401 = Problem & {
  code?: AddMediaServer401Code;
};
