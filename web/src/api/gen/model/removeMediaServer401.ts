import type { Problem } from "./problem.ts";
import type { RemoveMediaServer401Code } from "./removeMediaServer401Code.ts";

export type RemoveMediaServer401 = Problem & {
  code?: RemoveMediaServer401Code;
};
