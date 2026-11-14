import type { Problem } from "./problem.ts";
import type { RemoveMediaServer500Code } from "./removeMediaServer500Code.ts";

export type RemoveMediaServer500 = Problem & {
  code?: RemoveMediaServer500Code;
};
