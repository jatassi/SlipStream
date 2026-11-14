import type { Problem } from "./problem.ts";
import type { RemoveMediaServer404Code } from "./removeMediaServer404Code.ts";

export type RemoveMediaServer404 = Problem & {
  code?: RemoveMediaServer404Code;
};
