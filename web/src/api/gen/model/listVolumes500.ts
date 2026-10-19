import type { ListVolumes500Code } from "./listVolumes500Code.ts";
import type { Problem } from "./problem.ts";

export type ListVolumes500 = Problem & {
  code?: ListVolumes500Code;
};
